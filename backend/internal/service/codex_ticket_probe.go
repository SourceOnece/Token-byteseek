package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"github.com/TokenFlux/TokenRouter/internal/pkg/openai"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 诊断只记录数值、固定枚举和管理员自定义名称，绝不包含响应正文/票据/代理地址。
type CodexTicketDiagnostic struct {
	ProxyID        string     `json:"proxy_id"`
	ProxyName      string     `json:"proxy_name"`
	Attempt        int        `json:"attempt"`
	HTTPStatus     int        `json:"http_status,omitempty"`
	HeaderLength   int        `json:"header_length"`
	HeaderPresent  bool       `json:"header_present"`
	PrefixValid    bool       `json:"prefix_valid"`
	ResponseKind   string     `json:"response_kind,omitempty"`
	ErrorKind      string     `json:"error_kind,omitempty"`
	CompletionSeen bool       `json:"completion_seen,omitempty"`
	RetryNotBefore *time.Time `json:"retry_not_before,omitempty"`
}

// 状态接口再次收口枚举与范围，代理名称只从本次配置取，不信任缓存中的任意文案。
func safeCodexTicketDiagnostic(source *CodexTicketDiagnostic) *CodexTicketDiagnostic {
	if source == nil {
		return nil
	}
	d := *source
	d.ProxyName = ""
	if d.ProxyID != "legacy" {
		if _, err := uuid.Parse(d.ProxyID); err != nil {
			d.ProxyID = ""
		}
	}
	if d.Attempt < 1 || d.Attempt > 10 {
		d.Attempt = 0
	}
	if d.HTTPStatus < 100 || d.HTTPStatus > 599 {
		d.HTTPStatus = 0
	}
	if d.HeaderLength < 0 || d.HeaderLength > 1048576 {
		d.HeaderLength = 0
	}
	switch d.ResponseKind {
	case "sse", "json", "html", "other":
	default:
		d.ResponseKind = ""
	}
	switch d.ErrorKind {
	case "overloaded", "rate_limit", "quota", "auth", "invalid_request":
	default:
		d.ErrorKind = ""
	}
	return &d
}

func (s *CodexTicketService) probeAttempt(ctx context.Context, cfg *codexTicketConfig, account *Account, model, token, key string, proxy codexTicketProxy, attempt int) (ready, retry bool) {
	diagnostic := &CodexTicketDiagnostic{ProxyID: proxy.ID, ProxyName: proxy.Name, Attempt: attempt}
	proxyURL, err := s.cipher.Decrypt(proxy.Cipher)
	if err != nil || validateCodexHarvestProxy(proxyURL) != nil {
		s.recordObservation(ctx, key, "failed", "proxy_config", nil, diagnostic)
		return false, cfg.mode() == "rotate"
	}
	// 重试前重读凭据与资格；不得在配置变更/停调后继续拿旧快照请求。
	fresh, err := s.gateway.accountRepo.GetByID(ctx, account.ID)
	if err != nil || !codexTicketAccount(fresh) || !fresh.IsSchedulable() || !fresh.IsModelSupported(model) || fresh.GetOpenAIAccessToken() != token || codexTicketKey(cfg, fresh, model, token) != key {
		return false, false
	}
	probeCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	if s.gateway.concurrencyService != nil {
		slot, err := s.gateway.concurrencyService.AcquireAccountSlot(probeCtx, fresh.ID, fresh.Concurrency)
		if err != nil || slot == nil || !slot.Acquired {
			return false, false
		}
		defer slot.ReleaseFunc()
	}
	// 与回退前快照使用相同 input_text 数组，不再依赖字符串内容的宽松兼容。
	body, _ := json.Marshal(map[string]any{"model": model, "store": false, "stream": true, "instructions": "Reply with exactly: pong", "input": []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "input_text", "text": "ping"}}}}})
	probeCtx = WithHTTPUpstreamRedirectsDisabled(WithHTTPUpstreamProfile(probeCtx, HTTPUpstreamProfileOpenAIHarvest))
	req, err := http.NewRequestWithContext(probeCtx, http.MethodPost, chatgptCodexURL, bytes.NewReader(body))
	if err != nil {
		return false, false
	}
	req.Close = true
	req.Host = "chatgpt.com"
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("session_id", uuid.NewString())
	if resolveAndSetOpenAIChatGPTAccountHeaders(probeCtx, s.gateway.accountRepo, req.Header, fresh) != nil {
		s.recordObservation(ctx, key, "failed", "credential", nil, diagnostic)
		return false, false
	}
	ensureCodexIdentityHeaders(req.Header)
	enforceCodexIdentityHeaders(req.Header)
	minimum := "0.146.0"
	if model == "gpt-6-astra" {
		minimum = "0.153.4"
	}
	if CompareVersions(req.Header.Get("version"), minimum) < 0 {
		// 仅合成采集对齐固定快照的版本下限，保留更高的后台版本，不更改业务身份。
		req.Header.Set("version", minimum)
		req.Header.Set("user-agent", openai.CodexDefaultOriginator+"/"+minimum+" (Ubuntu 22.4.0; x86_64) xterm-256color")
		req.Header.Set("originator", openai.CodexDefaultOriginator)
	}
	s.recordObservation(ctx, key, "collecting", "", nil, diagnostic)
	state, reason := "failed", "network"
	var expires *time.Time
	defer func() {
		if ctx.Err() != nil && reason == "network" {
			reason = "cancelled"
		}
		s.recordObservation(ctx, key, state, reason, expires, diagnostic)
	}()
	resp, err := s.gateway.httpUpstream.Do(req, proxyURL, fresh.ID, fresh.Concurrency)
	if err != nil || resp == nil {
		return false, ctx.Err() == nil
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	ticket := codexTicketValue{State: extractOpenAICodexTurnState(resp.Header), ExpiresAt: time.Now().Add(time.Hour)}
	diagnostic.HTTPStatus = resp.StatusCode
	diagnostic.HeaderPresent = len(resp.Header.Values(openAICodexTurnStateHeader)) > 0
	diagnostic.HeaderLength = len(ticket.State)
	diagnostic.PrefixValid = strings.HasPrefix(ticket.State, "gAAAAA")
	switch strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])) {
	case "text/event-stream":
		diagnostic.ResponseKind = "sse"
	case "application/json":
		diagnostic.ResponseKind = "json"
	case "text/html":
		diagnostic.ResponseKind = "html"
	default:
		diagnostic.ResponseKind = "other"
	}
	if resp.StatusCode != http.StatusOK || !validCodexTicket(ticket) {
		reason = "upstream"
		if resp.StatusCode == http.StatusOK {
			state, reason = "missing", "invalid_ticket"
		}
		readTicketFailureDiagnostic(resp.Body, diagnostic, cancel)
		if diagnostic.ErrorKind != "" {
			state, reason = "failed", "upstream"
		}
		diagnostic.RetryNotBefore = codexTicketRetryNotBefore(resp.Header.Get("Retry-After"), resp.StatusCode, diagnostic.ErrorKind)
		if diagnostic.RetryNotBefore != nil {
			return false, false
		}
		// 授权/额度/明确限流不是通过切换 IP 无限重试可解决的问题。
		if resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 429 || diagnostic.ErrorKind == "auth" || diagnostic.ErrorKind == "quota" || diagnostic.ErrorKind == "rate_limit" || diagnostic.ErrorKind == "invalid_request" {
			return false, false
		}
		return false, ctx.Err() == nil
	}
	latest := s.enabledConfig()
	reason = "cancelled"
	if latest == nil || latest.Generation != cfg.Generation {
		return false, false
	}
	final, err := s.gateway.accountRepo.GetByID(ctx, account.ID)
	if err != nil || !codexTicketAccount(final) || !final.IsSchedulable() || !final.IsModelSupported(model) || final.GetOpenAIAccessToken() != token || codexTicketKey(cfg, final, model, token) != key {
		return false, false
	}
	raw, _ := json.Marshal(ticket)
	reason = "storage"
	encrypted, err := s.cipher.Encrypt(string(raw))
	if err != nil {
		return false, false
	}
	if s.cache.Set(ctx, key, encrypted, time.Hour) != nil {
		return false, false
	}
	state, reason, expires = "ready", "", &ticket.ExpiresAt
	return true, false
}

// 仅给后台采集退避，不修改账号调度；尊重服务端 Retry-After，避免换代理绕过明确限流。
func codexTicketRetryNotBefore(raw string, status int, kind string) *time.Time {
	now := time.Now().UTC()
	wait := time.Duration(0)
	if seconds, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); err == nil && seconds > 0 {
		if seconds > 86400 {
			seconds = 86400
		}
		wait = time.Duration(seconds) * time.Second
	} else if deadline, err := http.ParseTime(raw); err == nil && deadline.After(now) {
		wait = deadline.Sub(now)
	}
	if (status == 429 || kind == "rate_limit") && wait < time.Minute {
		wait = time.Minute
	}
	if wait <= 0 {
		return nil
	}
	if wait > 24*time.Hour {
		wait = 24 * time.Hour
	}
	deadline := now.Add(wait)
	return &deadline
}

// 只有未获有效票时读取少量失败数据，最多 8KiB/1.5 秒，任何原文都不持久化或返回。
func readTicketFailureDiagnostic(body io.ReadCloser, d *CodexTicketDiagnostic, cancel context.CancelFunc) {
	if body == nil {
		return
	}
	// 取消该次采集 HTTP 请求来解除网络 Read，不只依赖 Body.Close 的具体实现。
	timer := time.AfterFunc(1500*time.Millisecond, cancel)
	defer timer.Stop()
	scanner := bufio.NewScanner(io.LimitReader(body, 8192))
	var aggregate bytes.Buffer
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		aggregate.Write(line)
		aggregate.WriteByte('\n')
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		classifyTicketFailureJSON(line, d)
		if d.CompletionSeen || d.ErrorKind != "" {
			return
		}
	}
	// 兼容多行 JSON 错误信封，仍不保存或回传任何原始正文。
	classifyTicketFailureJSON(aggregate.Bytes(), d)
}

func classifyTicketFailureJSON(line []byte, d *CodexTicketDiagnostic) {
	if !gjson.ValidBytes(line) {
		return
	}
	typeName := gjson.GetBytes(line, "type").String()
	if typeName == "response.completed" || typeName == "response.done" {
		d.CompletionSeen = true
		return
	}
	for _, path := range []string{"error.code", "error.type", "response.error.code", "code"} {
		switch gjson.GetBytes(line, path).String() {
		case "server_is_overloaded", "server_overloaded":
			d.ErrorKind = "overloaded"
		case "rate_limit_exceeded", "rate_limit_error":
			d.ErrorKind = "rate_limit"
		case "insufficient_quota", "usage_limit_reached":
			d.ErrorKind = "quota"
		case "invalid_api_key", "authentication_error", "token_expired":
			d.ErrorKind = "auth"
		case "invalid_request_error":
			d.ErrorKind = "invalid_request"
		}
	}
	if d.ErrorKind == "" {
		paths := []string{"error.message", "response.error.message"}
		if typeName == "error" || d.HTTPStatus >= 400 {
			paths = append(paths, "message")
		}
		for _, path := range paths {
			if strings.Contains(strings.ToLower(gjson.GetBytes(line, path).String()), "our servers are currently overloaded") {
				d.ErrorKind = "overloaded"
			}
		}
	}
}
