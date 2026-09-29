package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
	"github.com/TokenFlux/TokenRouter/internal/gateway/ws"
	"github.com/TokenFlux/TokenRouter/internal/pkg/logredact"
	"github.com/TokenFlux/TokenRouter/internal/protocol/openai"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	upstreamopenai "github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func (s *OpenAIWebSocketExecutor) isOpenAIWSGeneratePrewarmEnabled() bool {
	return s != nil && s.Options != nil && s.Options.PrewarmGenerateEnabled
}

// performOpenAIWSGeneratePrewarm 在 WSv2 下执行可选的 generate=false 预热。
// 预热默认关闭，仅在配置开启后生效；失败时按可恢复错误回退到 HTTP。
func (s *OpenAIWebSocketExecutor) performOpenAIWSGeneratePrewarm(
	ctx context.Context,
	lease *upstreamopenai.WSConnLease,
	decision egress.OpenAIWSProtocolDecision,
	payload map[string]any,
	previousResponseID string,
	reqBody map[string]any,
	canonicalModel string,
	provider *gatewayprovider.ExecutionProvider,
	stateStore session.OpenAIWSStateStore,
	groupID int64,
) error {
	if s == nil {
		return nil
	}
	if lease == nil || provider == nil {
		gatewayprovider.LogOpenAIWSModeInfo("prewarm_skip reason=invalid_state has_lease=%v has_provider=%v", lease != nil, provider != nil)
		return nil
	}
	connID := strings.TrimSpace(lease.ConnID())
	if !s.isOpenAIWSGeneratePrewarmEnabled() {
		return nil
	}
	if decision.Transport != egress.OpenAIUpstreamTransportResponsesWebsocketV2 {
		gatewayprovider.LogOpenAIWSModeInfo(
			"prewarm_skip provider_id=%d conn_id=%s reason=transport_not_v2 transport=%s",
			provider.Record.ID,
			connID, gatewayprovider.NormalizeOpenAIWSLogValue(string(decision.Transport)),
		)
		return nil
	}
	if strings.TrimSpace(previousResponseID) != "" {
		gatewayprovider.LogOpenAIWSModeInfo(
			"prewarm_skip provider_id=%d conn_id=%s reason=has_previous_response_id previous_response_id=%s",
			provider.Record.ID,
			connID, gatewayprovider.TruncateOpenAIWSLogValue(previousResponseID, gatewayprovider.OpenAIWSIDValueMaxLen),
		)
		return nil
	}
	if lease.IsPrewarmed() {
		gatewayprovider.LogOpenAIWSModeInfo("prewarm_skip provider_id=%d conn_id=%s reason=already_prewarmed", provider.Record.ID, connID)
		return nil
	}
	if openai.NeedsToolContinuation(reqBody) {
		gatewayprovider.LogOpenAIWSModeInfo("prewarm_skip provider_id=%d conn_id=%s reason=tool_continuation", provider.Record.ID, connID)
		return nil
	}
	prewarmStart := time.Now()
	gatewayprovider.LogOpenAIWSModeInfo("prewarm_start provider_id=%d conn_id=%s", provider.Record.ID, connID)

	prewarmPayload := make(map[string]any, len(payload)+1)
	for k, v := range payload {
		prewarmPayload[k] = v
	}
	prewarmPayload["generate"] = false
	prewarmPayloadJSON := payloadAsJSONBytes(prewarmPayload)

	if err := lease.WriteJSONWithContextTimeout(ctx, prewarmPayload, s.openAIWSWriteTimeout()); err != nil {
		lease.MarkBroken()
		gatewayprovider.LogOpenAIWSModeInfo(
			"prewarm_write_fail provider_id=%d conn_id=%s cause=%s",
			provider.Record.ID,
			connID, gatewayprovider.TruncateOpenAIWSLogValue(err.Error(), gatewayprovider.OpenAIWSLogValueMaxLen),
		)
		return ws.WrapFallback("prewarm_write", err)
	}
	gatewayprovider.LogOpenAIWSModeInfo("prewarm_write_sent provider_id=%d conn_id=%s payload_bytes=%d", provider.Record.ID, connID, len(prewarmPayloadJSON))

	prewarmResponseID := ""
	prewarmEventCount := 0
	prewarmTerminalCount := 0
	for {
		message, readErr := lease.ReadMessageWithContextTimeout(ctx, s.openAIWSReadTimeout())
		if readErr != nil {
			lease.MarkBroken()
			closeStatus, closeReason := gatewayprovider.SummarizeOpenAIWSReadCloseError(readErr)
			gatewayprovider.LogOpenAIWSModeInfo(
				"prewarm_read_fail provider_id=%d conn_id=%s close_status=%s close_reason=%s cause=%s events=%d",
				provider.Record.ID,
				connID,
				closeStatus,
				closeReason, gatewayprovider.TruncateOpenAIWSLogValue(readErr.Error(), gatewayprovider.OpenAIWSLogValueMaxLen), prewarmEventCount,
			)
			return ws.WrapFallback("prewarm_"+gatewayprovider.ClassifyOpenAIWSReadFallbackReason(readErr), readErr)
		}

		eventType, eventResponseID, _ := openai.ParseWSEventEnvelope(message)
		if eventType == "" {
			continue
		}
		prewarmEventCount++
		if prewarmResponseID == "" && eventResponseID != "" {
			prewarmResponseID = eventResponseID
		}
		if prewarmEventCount <= openAIWSPrewarmEventLogHead || eventType == "error" || openai.IsWSTerminalEvent(eventType) {
			gatewayprovider.LogOpenAIWSModeInfo(
				"prewarm_event provider_id=%d conn_id=%s idx=%d type=%s bytes=%d",
				provider.Record.ID,
				connID,
				prewarmEventCount, gatewayprovider.TruncateOpenAIWSLogValue(eventType, gatewayprovider.OpenAIWSLogValueMaxLen), len(message),
			)
		}

		if eventType == "error" {
			errCodeRaw, errTypeRaw, errMsgRaw := openai.ParseWSErrorEventFields(message)
			errMsg := strings.TrimSpace(errMsgRaw)
			if errMsg == "" {
				errMsg = "OpenAI websocket prewarm error"
			}
			fallbackReason, canFallback := upstreamopenai.ClassifyWSErrorEventFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
			errCode, errType, errMessage := gatewayprovider.SummarizeOpenAIWSErrorEventFieldsFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
			gatewayprovider.LogOpenAIWSModeInfo(
				"prewarm_error_event provider_id=%d conn_id=%s idx=%d fallback_reason=%s can_fallback=%v err_code=%s err_type=%s err_message=%s",
				provider.Record.ID,
				connID,
				prewarmEventCount, gatewayprovider.TruncateOpenAIWSLogValue(fallbackReason, gatewayprovider.OpenAIWSLogValueMaxLen), canFallback,
				errCode,
				errType,
				errMessage,
			)
			lease.MarkBroken()
			statusCode := openAIWSErrorPolicyStatus(message)
			errorDecision := s.handleOpenAIWSErrorEventTransientFailure(
				ctx, provider, canonicalModel, lease.HandshakeHeaders(), message,
			)
			if errorDecision.ShouldReturnGenericError() {
				return ws.NewGenericPolicyError(statusCode)
			}
			if errorDecision.ShouldFailoverWithDefaults(gatewayprovider.ExecutionErrorPolicy(provider), statusCode, false, s.shouldFailoverOpenAIWSError(provider, statusCode, message)) {
				return gatewayprovider.NewOpenAIUpstreamFailure(
					statusCode,
					lease.HandshakeHeaders(),
					message,
					errMsg,
					errorDecision.RetryableOnSameProvider(gatewayprovider.ExecutionErrorPolicy(provider), statusCode),
				)
			}
			if canFallback {
				return ws.WrapFallback("prewarm_"+fallbackReason, errors.New(errMsg))
			}
			return ws.WrapFallback("prewarm_error_event", errors.New(errMsg))
		}

		if openai.IsWSTerminalEvent(eventType) {
			prewarmTerminalCount++
			break
		}
	}

	lease.MarkPrewarmed()
	if prewarmResponseID != "" && stateStore != nil {
		ttl := s.OpenAIHTTPResponseStickyTTL()
		gatewayprovider.LogOpenAIWSBindResponseProviderWarn(groupID, provider.Record.ID, prewarmResponseID, stateStore.BindResponseProvider(ctx, groupID, prewarmResponseID, provider.Record.ID, ttl))
		stateStore.BindResponseConn(prewarmResponseID, lease.ConnID(), ttl)
	}
	gatewayprovider.LogOpenAIWSModeInfo(
		"prewarm_done provider_id=%d conn_id=%s response_id=%s events=%d terminal_events=%d duration_ms=%d",
		provider.Record.ID,
		connID, gatewayprovider.TruncateOpenAIWSLogValue(prewarmResponseID, gatewayprovider.OpenAIWSIDValueMaxLen), prewarmEventCount,
		prewarmTerminalCount,
		time.Since(prewarmStart).Milliseconds(),
	)
	return nil
}

func payloadAsJSONBytes(payload map[string]any) []byte {
	if len(payload) == 0 {
		return []byte("{}")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return []byte("{}")
	}
	return body
}

func normalizeOpenAIWSTerminalEvent(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case "response.completed":
		return "response.completed"
	case "response.done":
		return "response.done"
	case "response.failed":
		return "response.failed"
	case "response.incomplete":
		return "response.incomplete"
	case "response.cancelled", "response.canceled":
		return "response.cancelled"
	default:
		return ""
	}
}

func openAIWSPayloadTransientStatus(payload []byte) int {
	if len(payload) == 0 {
		return 0
	}
	status := int(gjson.GetBytes(payload, "response.error.status_code").Int())
	if status == 0 {
		status = int(gjson.GetBytes(payload, "response.error.status").Int())
	}
	if status == 0 {
		status = int(gjson.GetBytes(payload, "error.status_code").Int())
	}
	if status == 0 {
		status = int(gjson.GetBytes(payload, "error.status").Int())
	}
	if gatewayprovider.IsTransientProviderFailure(status, payload) {
		return status
	}
	if status != 0 {
		return 0
	}
	code := strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "response.error.code").String()))
	errType := strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "response.error.type").String()))
	if code == "" {
		code = strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "error.code").String()))
	}
	if errType == "" {
		errType = strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "error.type").String()))
	}
	switch {
	case code == "server_is_overloaded", code == "slow_down":
		return http.StatusServiceUnavailable
	case strings.Contains(code, "server_error"),
		strings.Contains(code, "internal_error"),
		strings.Contains(code, "upstream_error"),
		strings.Contains(errType, "server_error"),
		strings.Contains(errType, "internal_error"),
		strings.Contains(errType, "upstream_error"):
		return http.StatusInternalServerError
	default:
		return 0
	}
}

// openAIWSErrorPolicyStatus 解析 WS 错误事件用于提供商策略的状态码。
// 事件显式携带状态码时必须原样保留，否则按既有 WS 错误类型映射，避免瞬态推断改变自定义规则的匹配值。
func openAIWSErrorPolicyStatus(payload []byte) int {
	if len(payload) == 0 {
		return 0
	}
	for _, path := range []string{
		"error.status_code",
		"error.status",
		"response.error.status_code",
		"response.error.status",
	} {
		status := int(gjson.GetBytes(payload, path).Int())
		if status >= http.StatusBadRequest && status <= 599 {
			return status
		}
	}
	codeRaw, errTypeRaw, _ := openai.ParseWSErrorEventFields(payload)
	if codeRaw == "" {
		codeRaw = strings.TrimSpace(gjson.GetBytes(payload, "response.error.code").String())
	}
	if errTypeRaw == "" {
		errTypeRaw = strings.TrimSpace(gjson.GetBytes(payload, "response.error.type").String())
	}
	return upstreamopenai.WSErrorHTTPStatusFromRaw(codeRaw, errTypeRaw)
}

// openAIWSTerminalPolicyDecision 保留终止事件类型及其提供商策略结果，
// 调用方必须在写给客户端前判断通用错误和故障转移。
type openAIWSTerminalPolicyDecision struct {
	TerminalEvent string
	StatusCode    int
	Decision      providercore.UpstreamErrorDecision
}

func (s *OpenAIWebSocketExecutor) handleOpenAIWSTerminalTransientFailure(ctx context.Context, provider *gatewayprovider.ExecutionProvider, canonicalModel string, headers http.Header, payload []byte) openAIWSTerminalPolicyDecision {
	eventType, _, _ := openai.ParseWSEventEnvelope(payload)
	result := openAIWSTerminalPolicyDecision{
		TerminalEvent: normalizeOpenAIWSTerminalEvent(eventType),
		Decision:      providercore.UpstreamErrorDecision{Policy: providercore.ErrorPolicyNone},
	}
	if result.TerminalEvent != "response.failed" {
		return result
	}
	result.StatusCode = openAIWSErrorPolicyStatus(payload)
	if result.StatusCode != 0 {
		if result.StatusCode == http.StatusTooManyRequests {
			headers = gatewayprovider.OpenAISemantic429Headers(provider, canonicalModel, headers)
		}
		result.Decision = s.applyOpenAIWSEventErrorPolicy(ctx, provider, canonicalModel, result.StatusCode, headers, payload)
	}
	return result
}

func (s *OpenAIWebSocketExecutor) handleOpenAIWSErrorEventTransientFailure(ctx context.Context, provider *gatewayprovider.ExecutionProvider, canonicalModel string, headers http.Header, payload []byte) providercore.UpstreamErrorDecision {
	eventType, _, _ := openai.ParseWSEventEnvelope(payload)
	if eventType != "error" {
		return providercore.UpstreamErrorDecision{Policy: providercore.ErrorPolicyNone}
	}
	status := openAIWSErrorPolicyStatus(payload)
	if status == http.StatusTooManyRequests {
		headers = gatewayprovider.OpenAISemantic429Headers(provider, canonicalModel, headers)
	}
	return s.applyOpenAIWSEventErrorPolicy(ctx, provider, canonicalModel, status, headers, payload)
}

// markOpenAIWSClientVisibleFailure 记录已经写给客户端的 WS 错误事件，避免把
// 已经完成故障转移的内部错误重复计入 Ops。
func markOpenAIWSClientVisibleFailure(c *gin.Context, eventType string, payload []byte) {
	eventType = strings.TrimSpace(eventType)
	if eventType != "error" && eventType != "response.failed" {
		return
	}
	prefix := "error"
	if eventType == "response.failed" {
		prefix = "response.error"
	}
	code := strings.TrimSpace(gjson.GetBytes(payload, prefix+".code").String())
	errType := strings.TrimSpace(gjson.GetBytes(payload, prefix+".type").String())
	message := strings.TrimSpace(gjson.GetBytes(payload, prefix+".message").String())
	if eventType == "response.failed" && code == "" && errType == "" && message == "" {
		prefix = "error"
		code = strings.TrimSpace(gjson.GetBytes(payload, prefix+".code").String())
		errType = strings.TrimSpace(gjson.GetBytes(payload, prefix+".type").String())
		message = strings.TrimSpace(gjson.GetBytes(payload, prefix+".message").String())
	}
	status := int(gjson.GetBytes(payload, prefix+".status_code").Int())
	if status == 0 {
		status = int(gjson.GetBytes(payload, prefix+".status").Int())
	}
	if status == 0 && eventType == "error" {
		status = int(gjson.GetBytes(payload, "status").Int())
	}
	if status == 0 {
		status = upstreamopenai.WSErrorHTTPStatusFromRaw(code, errType)
	}
	if errType == "" {
		errType = "upstream_error"
	}
	if code == "" {
		code = strings.ReplaceAll(eventType, ".", "_")
	}
	if message == "" {
		message = "upstream websocket request failed"
	}
	MarkOpsStreamFailure(c, errType, code, message, status)
}

// handleOpenAIWSFailureProviderSideEffects 将 WS 错误事件映射到提供商健康策略，
// 返回值用于成对的 error/response.failed 事件去重。
func (s *OpenAIWebSocketExecutor) handleOpenAIWSFailureProviderSideEffects(ctx context.Context, provider *gatewayprovider.ExecutionProvider, canonicalModel string, headers http.Header, payload []byte) bool {
	message := upstreamopenai.ExtractOpenAISSEErrorMessage(payload)
	status := upstreamopenai.OpenAIStreamFailureStatus(payload, message)
	switch status {
	case http.StatusUnauthorized, http.StatusTooManyRequests, 529:
		s.Output.TerminalProviderEffects(nil, provider, payload, message, headers, canonicalModel)
		return true
	case http.StatusForbidden:
		if !upstreamopenai.OpenAIStream403ProviderFailure(payload, message) {
			return false
		}
		s.Output.TerminalProviderEffects(nil, provider, payload, message, headers, canonicalModel)
		return true
	}
	status = openAIWSPayloadTransientStatus(payload)
	if status == 0 {
		return false
	}
	gatewayprovider.ApplyOpenAIResponseHealth(ctx, s.Output.Health, provider, status, headers, payload, false, canonicalModel)
	return true
}

func (s *OpenAIWebSocketExecutor) handleOpenAIWSDialTransientFailure(ctx context.Context, provider *gatewayprovider.ExecutionProvider, canonicalModel string, err error) providercore.UpstreamErrorDecision {
	var dialErr *upstreamopenai.WSDialError
	if !errors.As(err, &dialErr) || dialErr == nil {
		return providercore.UpstreamErrorDecision{Policy: providercore.ErrorPolicyNone}
	}
	return s.applyOpenAIWSEventErrorPolicy(ctx, provider, canonicalModel, dialErr.StatusCode, dialErr.ResponseHeaders, dialErr.ResponseBody)
}

// applyOpenAIWSEventErrorPolicy 将握手和事件错误接入统一提供商策略。
// 请求级错误保持原样，响应尚未输出时由调用方依据返回决策决定是否故障转移。
func (s *OpenAIWebSocketExecutor) applyOpenAIWSEventErrorPolicy(
	ctx context.Context,
	provider *gatewayprovider.ExecutionProvider,
	canonicalModel string,
	statusCode int,
	headers http.Header,
	payload []byte,
) providercore.UpstreamErrorDecision {
	if statusCode == 0 || gatewayprovider.OpenAIWSHTTPBridgeRequestScopedError(provider, statusCode, upstream.ExtractErrorMessage(payload), payload) {
		return providercore.UpstreamErrorDecision{Policy: providercore.ErrorPolicyNone}
	}
	if provider != nil && provider.Record.Platform == capability.PlatformGrok {
		return gatewayprovider.ApplyGrokExecutionHealth(ctx, s.Output.GrokHealth, provider, statusCode, headers, payload, "", canonicalModel)
	}
	return gatewayprovider.ApplyOpenAIResponseHealth(ctx, s.Output.Health, provider, statusCode, headers, payload, false, canonicalModel)
}

// shouldFailoverOpenAIWSError 使用对应平台的 HTTP 错误分类作为 WS 握手和事件错误的默认切号规则。
func (s *OpenAIWebSocketExecutor) shouldFailoverOpenAIWSError(provider *gatewayprovider.ExecutionProvider, statusCode int, payload []byte) bool {
	if statusCode == 0 {
		return false
	}
	if provider != nil && provider.Record.Platform == capability.PlatformGrok {
		return gatewayprovider.ShouldFailoverGrokResponse(statusCode, payload)
	}
	upstreamMsg := logredact.SanitizeUpstreamQueries(strings.TrimSpace(upstream.ExtractErrorMessage(payload)))
	return gatewayprovider.ShouldFailoverOpenAIResponse(statusCode, upstreamMsg, payload)
}

// openAIWSGenericPolicyCloseError 在 WS 入站尚未输出时用统一文案终止连接。
func openAIWSGenericPolicyCloseError(statusCode int) error {
	return NewOpenAIWSClientCloseError(
		coderws.StatusInternalError,
		"Upstream gateway error", ws.NewGenericPolicyError(statusCode),
	)
}
