package service

// 双链路完成校验参考STATE Kit（LGPL-3.0），固定fdda6486；本地增加JSON、诊断及账号配置边界。
import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"time"
	"unicode"

	"github.com/tidwall/gjson"
)

type ticketCompletion struct {
	NetworkKind string
	Model       string
	Complete    bool
	Reason      string
	ErrorKind   string
}

// 诊断模型仅限短文本，不记录任意响应正文或控制字符。
func safeTicketResponseModel(model string) string {
	model = strings.TrimSpace(model)
	if len([]rune(model)) > 200 || strings.IndexFunc(model, unicode.IsControl) >= 0 {
		return ""
	}
	return model
}

// 完成事件才是校验依据；delta、[DONE]、提前断流或缺模型不冒充匹配成功。
func parseTicketCompletion(body io.Reader, expected string) ticketCompletion {
	if body == nil {
		return ticketCompletion{Reason: "incomplete_response"}
	}
	limited := &io.LimitedReader{R: body, N: (2 << 20) + 1}
	reader := bufio.NewReader(limited)
	for {
		b, err := reader.Peek(1)
		if err != nil {
			return incompleteTicketRead(err)
		}
		if !strings.ContainsRune(" \t\r\n", rune(b[0])) {
			break
		}
		_, _ = reader.ReadByte()
	}
	first, _ := reader.Peek(1)
	if first[0] == '{' {
		raw, err := io.ReadAll(reader)
		if err != nil {
			return incompleteTicketRead(err)
		}
		if limited.N <= 0 {
			return ticketCompletion{Reason: "incomplete_response"}
		}
		return inspectTicketCompletion(raw, "", expected, true)
	}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	event := ""
	var data strings.Builder
	check := func() ticketCompletion {
		if data.Len() == 0 {
			return ticketCompletion{}
		}
		return inspectTicketCompletion([]byte(data.String()), event, expected, false)
	}
	for scanner.Scan() {
		line := scanner.Text()
		if limited.N <= 0 {
			return ticketCompletion{Reason: "incomplete_response"}
		}
		if line == "" {
			result := check()
			if result.Complete || result.Reason != "" {
				return result
			}
			data.Reset()
			event = ""
			continue
		}
		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(line[6:])
		}
		if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimSpace(line[5:]))
			if data.Len() > 1<<20 {
				return ticketCompletion{Reason: "incomplete_response"}
			}
		}
	}
	if scanner.Err() != nil {
		return incompleteTicketRead(scanner.Err())
	}
	if limited.N <= 0 {
		return ticketCompletion{Reason: "incomplete_response"}
	}
	result := check()
	if result.Complete || result.Reason != "" {
		return result
	}
	return ticketCompletion{Reason: "incomplete_response"}
}

// 区分正常缺少终态和读取连接错误；仍判不完整，不放宽发布条件。
func incompleteTicketRead(err error) ticketCompletion {
	value := ticketCompletion{Reason: "incomplete_response"}
	if err != nil && err != io.EOF && err != bufio.ErrTooLong {
		value.NetworkKind = ticketNetworkKind(err)
	}
	return value
}

func inspectTicketCompletion(raw []byte, event, expected string, jsonBody bool) ticketCompletion {
	if !gjson.ValidBytes(raw) {
		return ticketCompletion{Reason: "incomplete_response"}
	}
	root := gjson.ParseBytes(raw)
	// HTTP 200并不代表SSE成功；复用原分类，保留额度/认证/限流拒绝的语义。
	diagnostic := &CodexTicketDiagnostic{}
	classifyTicketFailureJSON(raw, diagnostic)
	if diagnostic.ErrorKind != "" {
		return ticketCompletion{Reason: "upstream", ErrorKind: diagnostic.ErrorKind}
	}
	typ := root.Get("type").String()
	if typ == "" {
		typ = event
	}
	if typ == "error" || typ == "response.failed" || typ == "response.incomplete" {
		return ticketCompletion{Reason: "incomplete_response"}
	}
	response := root
	if typ == "response.completed" {
		response = root.Get("response")
	} else if !jsonBody || root.Get("object").String() != "response" || root.Get("status").String() != "completed" {
		return ticketCompletion{}
	}
	model := safeTicketResponseModel(response.Get("model").String())
	status := response.Get("status").String()
	if model == "" || (status != "" && status != "completed") {
		return ticketCompletion{Model: model, Reason: "incomplete_response"}
	}
	result := ticketCompletion{Model: model, Complete: true}
	if model != expected {
		result.Reason = "model_mismatch"
	}
	return result
}

// 复验复用账号现有业务代理，不从采集代理偷偷回退；无代理边界按新模式的显式要求处理。
func ticketBusinessRoute(account *Account) (string, bool) {
	if account == nil || account.ProxyID == nil || account.Proxy == nil || account.Proxy.ID != *account.ProxyID || !account.Proxy.IsActive() || account.Proxy.IsExpired(time.Now()) {
		return "", false
	}
	raw := account.Proxy.URL()
	if validateCodexHarvestProxy(raw) != nil {
		return "", false
	}
	return raw, true
}

func ticketBusinessFingerprint(account *Account) string {
	raw, ok := ticketBusinessRoute(account)
	if !ok {
		return ""
	}
	encoded, _ := json.Marshal([]any{account.ID, account.ProxyID, raw})
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func validTicketForAccount(value codexTicketValue, cfg *codexTicketConfig, account *Account) bool {
	if !validCodexTicket(value, cfg.targetLength()) {
		return false
	}
	if !cfg.VerifiedFlow {
		return true
	}
	fingerprint := ticketBusinessFingerprint(account)
	return fingerprint != "" && value.Verified && value.BusinessFingerprint == fingerprint
}

// 仅新模式改为逐轮HTTP上游；不借票据开关开启被全局或账号明确禁用的WS。
func (s *OpenAIGatewayService) shouldBridgeVerifiedTicketAccount(account *Account, decision OpenAIWSProtocolDecision) bool {
	if s == nil || !codexTicketAccount(account) {
		return false
	}
	if decision.Transport != OpenAIUpstreamTransportResponsesWebsocketV2 && decision.Reason != "ws_v2_mode_http_bridge" {
		return false
	}
	tickets := s.codexTickets.Load()
	if tickets == nil {
		return false
	}
	cfg, _ := tickets.routingConfig(account.ID)
	return cfg != nil && cfg.VerifiedFlow
}
