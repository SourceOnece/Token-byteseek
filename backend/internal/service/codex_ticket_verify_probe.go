package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

// 管理日志仅记录两阶段安全摘要；不保存STATE、令牌、代理URL或响应正文。
type CodexTicketValidationStage struct {
	Name                 string `json:"name"`
	RequestModel         string `json:"request_model"`
	ResponseModel        string `json:"response_model,omitempty"`
	HTTPStatus           int    `json:"http_status,omitempty"`
	StateLength          int    `json:"state_length"`
	TargetLength         int    `json:"target_length,omitempty"`
	DegradedSignalLength int    `json:"degraded_signal_length,omitempty"`
	Complete             bool   `json:"complete"`
	Reason               string `json:"reason,omitempty"`
}

func safeTicketValidationReason(reason string) string {
	switch reason {
	case "business_proxy", "network", "upstream", "invalid_ticket", "incomplete_response", "model_mismatch", "length_signal", "account_changed", "timeout", "transport_timeout", "task_timeout", "round_timeout", "client_disconnected", "config_unavailable", "config_disabled", "lease_lost", "storage", "service_stopped":
		return reason
	}
	return ""
}

// 候选与业务出口复验共用账号单次预算；两者完整成功才允许调用方发布原候选票。
// @project-doc docs/interfaces/codex_ticket.md#verified_flow
func (s *CodexTicketService) validateTicketChain(ctx context.Context, cancel context.CancelFunc, cfg *codexTicketConfig, a *Account, model string, original *http.Request, body []byte, candidate *http.Response, diagnostic *CodexTicketDiagnostic) (bool, bool, string) {
	check := func(name string, resp *http.Response) (bool, bool, string) {
		diagnostic.Phase = name
		stage := CodexTicketValidationStage{Name: name, RequestModel: model, TargetLength: cfg.targetLength(), DegradedSignalLength: cfg.DegradedSignalLength}
		defer func() { diagnostic.Stages = append(diagnostic.Stages, stage) }()
		if resp == nil {
			diagnostic.HTTPStatus, diagnostic.HeaderLength = 0, 0
			diagnostic.HeaderPresent, diagnostic.PrefixValid = false, false
			stage.Reason = "network"
			return false, true, stage.Reason
		}
		stage.HTTPStatus = resp.StatusCode
		state := extractOpenAICodexTurnState(resp.Header)
		stage.StateLength = len(state)
		diagnostic.HTTPStatus = resp.StatusCode
		diagnostic.HeaderLength = len(state)
		diagnostic.HeaderPresent = len(resp.Header.Values(openAICodexTurnStateHeader)) > 0
		diagnostic.PrefixValid = strings.HasPrefix(state, "gAAAAA")
		if resp.StatusCode != http.StatusOK {
			stage.Reason = "upstream"
			diagnostic.HTTPStatus = resp.StatusCode
			readTicketFailureDiagnostic(resp.Body, diagnostic, cancel)
			return false, ticketFailureCanRetry(resp, diagnostic), stage.Reason
		}
		completion := parseTicketCompletion(resp.Body, model)
		diagnostic.NetworkKind = completion.NetworkKind
		stage.ResponseModel, stage.Complete = completion.Model, completion.Complete
		if completion.ErrorKind != "" {
			diagnostic.ErrorKind = completion.ErrorKind
			stage.Reason = "upstream"
			return false, ticketFailureCanRetry(resp, diagnostic), stage.Reason
		}
		ticket := codexTicketValue{State: state, ExpiresAt: time.Now().Add(time.Minute)}
		if cfg.DegradedSignalLength > 0 && validCodexTicket(ticket, cfg.DegradedSignalLength) {
			diagnostic.DegradedSignal = true
			stage.Reason = "length_signal"
			return false, true, stage.Reason
		}
		if name == "harvest" && !validCodexTicket(ticket, cfg.targetLength()) {
			stage.Reason = "invalid_ticket"
			return false, true, stage.Reason
		}
		// 明确长度结论仍按原顺序优先；仅无明确结论的不完整响应细分为超时。
		if !completion.Complete && ctx.Err() != nil {
			stage.Reason = ticketContextReason(ctx)
			return false, stage.Reason == "timeout", stage.Reason
		}
		if !completion.Complete || completion.Reason != "" {
			stage.Reason = completion.Reason
			if stage.Reason == "" {
				stage.Reason = "incomplete_response"
			}
			return false, true, stage.Reason
		}
		return true, false, ""
	}
	if ok, retry, reason := check("harvest", candidate); !ok {
		return false, retry, reason
	}
	// 完成事件后立即关闭候选连接，再发验证请求，不在此重新取动态代理。
	if candidate.Body != nil {
		_ = candidate.Body.Close()
	}
	if reason := ticketContextReason(ctx); reason != "" {
		return false, reason == "timeout", reason
	}
	if !s.ticketConfigCurrent(cfg) {
		return false, false, s.ticketConfigurationReason(cfg)
	}
	live, err := s.readTicketAccount(ctx, a.ID)
	if err != nil {
		if why := ticketContextReason(ctx); why != "" {
			return false, why == "timeout", why
		}
		return false, false, "storage"
	}
	if why, _ := ticketCollectionPause(live); why != "" {
		return false, false, why
	}
	if !live.IsModelSupported(model) {
		return false, false, "model_unsupported"
	}
	if live.GetOpenAIAccessToken() != a.GetOpenAIAccessToken() || codexTicketKey(cfg, live, model, live.GetOpenAIAccessToken()) != codexTicketKey(cfg, a, model, a.GetOpenAIAccessToken()) {
		return false, false, "account_changed"
	}
	proxy, valid := ticketBusinessRoute(live)
	if !valid {
		return false, false, "business_proxy"
	}
	req := original.Clone(ctx)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.Header.Set(openAICodexTurnStateHeader, extractOpenAICodexTurnState(candidate.Header))
	// 仅复验请求使用账号业务代理和TLS模板，不改业务流的代理、并发或返回内容。
	diagnostic.Phase = "verify"
	response, err := s.gateway.httpUpstream.DoWithTLS(req, proxy, live.ID, live.Concurrency, s.gateway.resolveOpenAITLSProfile(live))
	if err != nil {
		diagnostic.HTTPStatus, diagnostic.HeaderLength = 0, 0
		diagnostic.HeaderPresent, diagnostic.PrefixValid = false, false
		reason := ticketNetworkReason(ctx, err, diagnostic)
		diagnostic.Stages = append(diagnostic.Stages, CodexTicketValidationStage{Name: "verify", RequestModel: model, TargetLength: cfg.targetLength(), DegradedSignalLength: cfg.DegradedSignalLength, Reason: reason})
		return false, reason == "timeout" || ctx.Err() == nil, reason
	}
	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}
	return check("verify", response)
}
