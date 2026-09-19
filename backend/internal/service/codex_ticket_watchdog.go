package service

// 守护收据与有界响应观察参考 wangyunjeff/sub2api-state-kit（LGPL-3.0），
// 固定来源74d51079；本地改用加密Redis原子废票，并保留原生WS与集群租约。

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

// 守护摘要只保存固定枚举和时间；不保存响应正文、STATE、Token或代理地址。
type CodexTicketWatchdogStatus struct {
	Mode      string    `json:"mode"`
	Count     int64     `json:"count"`
	Reason    string    `json:"reason,omitempty"`
	Action    string    `json:"action,omitempty"`
	CheckedAt time.Time `json:"checked_at,omitempty"`
}
type CodexTicketWatchdogCache interface {
	ObserveTicket(context.Context, string, string, string, bool, time.Time) (bool, error)
}
type codexTicketReceipt struct {
	s                   *CodexTicketService
	cfg                 *codexTicketConfig
	key, encoded, model string
	stateHash           [32]byte
}

// 重连请求可能被既有回合状态更新；只记录确实发送了本平台原票据的握手。
func (r *codexTicketReceipt) forHeaders(headers http.Header) *codexTicketReceipt {
	if r == nil || sha256.Sum256([]byte(headers.Get(openAICodexTurnStateHeader))) != r.stateHash {
		return nil
	}
	return r
}

type ticketReceiptContextKey struct{}

func (s *CodexTicketService) ApplyRequest(ctx context.Context, account *Account, model string, req *http.Request) error {
	receipt, err := s.applyWithReceipt(ctx, account, model, req.Header)
	if err == nil && receipt != nil {
		*req = *req.WithContext(context.WithValue(req.Context(), ticketReceiptContextKey{}, receipt))
	}
	return err
}

// 只旁观已注入本平台票据的成功响应，保持原始字节/读取次数/业务错误原样。
func (s *CodexTicketService) ObserveResponse(req *http.Request, resp *http.Response) {
	if req == nil || resp == nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return
	}
	r, _ := req.Context().Value(ticketReceiptContextKey{}).(*codexTicketReceipt)
	if r == nil {
		return
	}
	r.observeHeader(resp.Header)
	if resp.Body != nil {
		resp.Body = &ticketWatchdogBody{ReadCloser: resp.Body, receipt: r}
	}
}

func (r *codexTicketReceipt) observeHeader(header http.Header) {
	if r == nil || r.cfg.DegradedSignalLength == 0 {
		return
	}
	value := extractOpenAICodexTurnState(header)
	if len(value) == r.cfg.DegradedSignalLength && strings.HasPrefix(value, "gAAAAA") && !strings.ContainsAny(value, "\r\n\x00") {
		r.signal("length_signal")
	}
}
func (r *codexTicketReceipt) observeJSON(raw []byte, eventName string, model string) {
	if r == nil || model != r.model || len(raw) > 1024*1024 || !gjson.ValidBytes(raw) {
		return
	}
	root := gjson.ParseBytes(raw)
	typ := root.Get("type").String()
	completed := typ == "response.completed" || (typ == "" && eventName == "response.completed")
	response := root
	if completed {
		response = root.Get("response")
	} else if typ != "" || root.Get("object").String() != "response" {
		return
	}
	status := response.Get("status").String()
	if status != "completed" && !(completed && status == "") {
		return
	}
	actual := response.Get("model")
	if actual.Type == gjson.String && strings.TrimSpace(actual.String()) != "" && strings.TrimSpace(actual.String()) != model {
		r.signal("model_mismatch")
	}
}

func (r *codexTicketReceipt) signal(reason string) {
	if r == nil || !r.s.ticketConfigCurrent(r.cfg) {
		return
	}
	cache, ok := r.s.cache.(CodexTicketWatchdogCache)
	if !ok {
		return
	}
	mode := ticketWatchdogMode(r.cfg.WatchdogMode)
	if mode == "off" {
		return
	}
	revoke := mode == "recover" || (mode == "recover_length" && reason == "length_signal") || (mode == "recover_model" && reason == "model_mismatch")
	// 单次只访问Redis，严格有界，不同步调用上游或查询账号数据库。
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	changed, err := cache.ObserveTicket(ctx, r.key, r.encoded, reason, revoke, time.Now().UTC())
	if err != nil || !changed || !revoke {
		return
	}
	// 唤醒原有后台轮次，沿用集群租约、并发槽及Retry-After，不另起无界任务。
	r.s.watchdogWake.Store(true)
}

// 有界增量解析：每个JSON或SSE事件最多1MiB。过大/损坏/未完成内容只跳过观察。
type ticketWatchdogBody struct {
	io.ReadCloser
	receipt                      *codexTicketReceipt
	mu                           sync.Mutex
	mode                         byte
	line, event                  []byte
	name                         string
	overflow, skipLine, finished bool
}

func (b *ticketWatchdogBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.finished {
		if n > 0 {
			b.observe(p[:n])
		}
		if err == io.EOF {
			b.finish()
		}
	}
	return n, err
}
func (b *ticketWatchdogBody) Close() error {
	err := b.ReadCloser.Close()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.finish()
	return err
}
func (b *ticketWatchdogBody) finish() {
	if b.finished {
		return
	}
	b.finished = true
	if b.mode == 'j' && !b.overflow {
		b.receipt.observeJSON(b.line, "", b.receipt.model)
	}
	if b.mode == 's' {
		if !b.skipLine && len(b.line) > 0 {
			b.parseLine(b.line)
		}
		b.flush()
	}
}
func (b *ticketWatchdogBody) observe(p []byte) {
	if b.mode == 0 {
		v := bytes.TrimSpace(p)
		if len(v) == 0 {
			return
		}
		b.mode = 's'
		if v[0] == '{' {
			b.mode = 'j'
		}
	}
	if b.mode == 'j' {
		if !b.overflow && len(b.line)+len(p) <= 1024*1024 {
			b.line = append(b.line, p...)
		} else {
			b.overflow = true
			b.line = nil
		}
		return
	}
	for len(p) > 0 {
		i := bytes.IndexByte(p, '\n')
		part := p
		if i >= 0 {
			part = p[:i]
		}
		if !b.skipLine {
			if len(b.line)+len(part) > 1024*1024 {
				b.line = nil
				b.skipLine = true
				b.overflow = true
			} else {
				b.line = append(b.line, part...)
			}
		}
		if i < 0 {
			return
		}
		if !b.skipLine {
			b.parseLine(b.line)
		}
		b.line = b.line[:0]
		b.skipLine = false
		p = p[i+1:]
	}
}
func (b *ticketWatchdogBody) parseLine(line []byte) {
	line = bytes.TrimSuffix(line, []byte{'\r'})
	if len(line) == 0 {
		b.flush()
		return
	}
	if bytes.HasPrefix(line, []byte("event:")) {
		b.name = ""
		if strings.TrimSpace(string(line[6:])) == "response.completed" {
			b.name = "response.completed"
		}
	}
	if !b.overflow && bytes.HasPrefix(line, []byte("data:")) {
		v := bytes.TrimPrefix(line[5:], []byte{' '})
		if len(b.event)+len(v)+1 > 1024*1024 {
			b.event = nil
			b.overflow = true
		} else {
			b.event = append(b.event, v...)
			b.event = append(b.event, '\n')
		}
	}
}
func (b *ticketWatchdogBody) flush() {
	if !b.overflow {
		b.receipt.observeJSON(b.event, b.name, b.receipt.model)
	}
	b.event = b.event[:0]
	b.name = ""
	b.overflow = false
}

func safeTicketWatchdog(raw, mode string) *CodexTicketWatchdogStatus {
	v := &CodexTicketWatchdogStatus{Mode: ticketWatchdogMode(mode)}
	var stored CodexTicketWatchdogStatus
	if json.Unmarshal([]byte(raw), &stored) == nil && stored.Count >= 0 && (stored.Reason == "length_signal" || stored.Reason == "model_mismatch") && (stored.Action == "observed" || stored.Action == "revoked") {
		stored.Mode = v.Mode
		return &stored
	}
	return v
}
