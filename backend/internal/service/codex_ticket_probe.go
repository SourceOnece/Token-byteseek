package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	Stages         []CodexTicketValidationStage `json:"stages,omitempty"`
	Scheduling     string                       `json:"scheduling,omitempty"`
	DegradedSignal bool                         `json:"degraded_signal,omitempty"`
	ProxyID        string                       `json:"proxy_id"`
	ProxyName      string                       `json:"proxy_name"`
	Attempt        int                          `json:"attempt"`
	HTTPStatus     int                          `json:"http_status,omitempty"`
	HeaderLength   int                          `json:"header_length"`
	HeaderPresent  bool                         `json:"header_present"`
	PrefixValid    bool                         `json:"prefix_valid"`
	ResponseKind   string                       `json:"response_kind,omitempty"`
	ErrorKind      string                       `json:"error_kind,omitempty"`
	CompletionSeen bool                         `json:"completion_seen,omitempty"`
	RetryNotBefore *time.Time                   `json:"retry_not_before,omitempty"`
}

// 状态接口再次收口枚举与范围，代理名称只从本次配置取，不信任缓存中的任意文案。
func safeCodexTicketDiagnostic(source *CodexTicketDiagnostic) *CodexTicketDiagnostic {
	if source == nil {
		return nil
	}
	d := *source
	d.Stages = nil
	for i, stage := range source.Stages {
		if i >= 2 {
			break
		}
		if stage.Name != "harvest" && stage.Name != "verify" {
			continue
		}
		stage.RequestModel = safeTicketResponseModel(stage.RequestModel)
		stage.ResponseModel = safeTicketResponseModel(stage.ResponseModel)
		if stage.HTTPStatus < 100 || stage.HTTPStatus > 599 {
			stage.HTTPStatus = 0
		}
		if stage.StateLength < 0 || stage.StateLength > 8192 {
			stage.StateLength = 0
		}
		stage.Reason = safeTicketValidationReason(stage.Reason)
		d.Stages = append(d.Stages, stage)
	}
	switch d.Scheduling {
	case "enabled", "disabled", "already_on", "already_off", "stale", "failed":
	default:
		d.Scheduling = ""
	}
	d.ProxyName = ""
	if d.ProxyID != "legacy" && d.ProxyID != "account" && d.ProxyID != "provider" {
		if _, err := uuid.Parse(d.ProxyID); err != nil {
			d.ProxyID = ""
		}
	}
	if d.Attempt < 1 {
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

func (s *CodexTicketService) probeAttempt(ctx context.Context, cfg *codexTicketConfig, account *Account, model, token, key string, proxy codexTicketProxy, attempt int, observers ...func(CodexTicketAttempt)) (ready, retry bool) {
	diagnostic := &CodexTicketDiagnostic{ProxyID: proxy.ID, ProxyName: proxy.Name, Attempt: attempt}
	started := time.Now().UTC()
	state, reason := "skipped", "account_changed"
	var expires *time.Time
	// 手动日志在统一出口收集，自动采集不增加历史/IP 请求。
	defer func() {
		if ctx.Err() != nil && !ready {
			state, reason = "cancelled", "cancelled"
		}
		if state == "failed" || state == "missing" || ready {
			s.recordTicketCollection(ctx, cfg, key, ready, reason)
		}
		if len(observers) == 0 {
			s.recordLatest(ctx, cfg, key, "auto", CodexTicketAttempt{Status: state, Reason: reason, FinishedAt: time.Now().UTC(), Diagnostic: diagnostic})
		}
		for _, observer := range observers {
			observer(CodexTicketAttempt{usedProxy: proxy, Status: state, Reason: reason, Attempt: attempt, StartedAt: started, FinishedAt: time.Now().UTC(), DurationMS: time.Since(started).Milliseconds(), Diagnostic: safeCodexTicketDiagnostic(diagnostic), ExpiresAt: expires})
		}
	}()
	if !s.ticketConfigCurrent(cfg) {
		return false, false
	}
	policyRaw, e := s.cache.Get(ctx, "collection:"+key)
	if e != nil {
		state, reason = "failed", "storage"
		return false, false
	}
	if ticketCollectionState(policyRaw).CooldownUntil != nil {
		state, reason = "skipped", "cooldown"
		return false, true
	}
	// 账号级采集槽在取号前获得；45秒租约覆盖8秒取号及25秒请求，不与业务并发槽混用。
	release, err := s.acquireTicketCollectionSlot(ctx, cfg, account.ID)
	if err != nil {
		state, reason = "failed", "storage"
		return false, false
	}
	defer release()
	// 先复核凭据与并发，再请求取号服务；排队/停调不会白白消耗代理额度和尝试计数。
	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	fresh, err := s.gateway.accountRepo.GetByID(readCtx, account.ID)
	readCancel()
	if err != nil || !s.ticketConfigCurrent(cfg) || !codexTicketCollectionAllowed(ctx, fresh) || !fresh.IsModelSupported(model) || fresh.GetOpenAIAccessToken() != token || codexTicketKey(cfg, fresh, model, token) != key {
		return false, false
	}
	if cfg.VerifiedFlow && ticketBusinessFingerprint(fresh) == "" {
		state, reason = "failed", "business_proxy"
		return false, false
	}
	probeCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	if s.gateway.concurrencyService != nil {
		slot, err := s.gateway.concurrencyService.AcquireAccountSlot(probeCtx, fresh.ID, fresh.Concurrency)
		if err != nil || slot == nil || !slot.Acquired {
			reason = "concurrency_busy"
			return false, false
		}
		defer slot.ReleaseFunc()
	}
	if len(observers) == 0 {
		attempt, err = s.nextTicketAttempt(ctx, key, cfg.attempts(), attempt)
		if err != nil {
			state, reason = "failed", "storage"
			return false, false
		}
		if attempt == 0 {
			state, reason = "skipped", "attempt_limit"
			return false, false
		}
		diagnostic.Attempt = attempt
	}
	proxy, err = s.resolveTicketAttemptProxy(probeCtx, cfg, proxy)
	if err != nil {
		state, reason = "failed", "proxy_provider"
		var rejected *ticketProviderRejection
		if errors.As(err, &rejected) {
			diagnostic.RetryNotBefore = rejected.retryAt
		}
		s.recordObservation(ctx, key, state, reason, nil, diagnostic)
		if rejected != nil && (rejected.status == 401 || rejected.status == 403 || rejected.status == 429) {
			return false, false
		}
		return false, ctx.Err() == nil
	}
	proxyURL, err := s.cipher.Decrypt(proxy.Cipher)
	if err != nil || validateCodexHarvestProxy(proxyURL) != nil {
		state, reason = "failed", "proxy_config"
		s.recordObservation(ctx, key, "failed", "proxy_config", nil, diagnostic)
		return false, cfg.mode() == "rotate"
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
		state, reason = "failed", "credential"
		s.recordObservation(ctx, key, "failed", "credential", nil, diagnostic)
		return false, false
	}
	ensureCodexIdentityHeaders(req.Header)
	enforceCodexIdentityHeaders(req.Header)
	minimum := "0.146.0"
	if strings.Contains(strings.ToLower(model), "gpt-6") || strings.Contains(strings.ToLower(model), "astra") {
		minimum = "0.153.4"
	}
	if CompareVersions(req.Header.Get("version"), minimum) < 0 {
		// 仅合成采集对齐固定快照的版本下限，保留更高的后台版本，不更改业务身份。
		req.Header.Set("version", minimum)
		req.Header.Set("user-agent", openai.CodexDefaultOriginator+"/"+minimum+" (Ubuntu 22.4.0; x86_64) xterm-256color")
		req.Header.Set("originator", openai.CodexDefaultOriginator)
	}
	s.recordObservation(ctx, key, "collecting", "", nil, diagnostic)
	state, reason = "failed", "network"
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
	if cfg.VerifiedFlow {
		ok, canRetry, why := s.validateTicketChain(probeCtx, cancel, cfg, fresh, model, req, body, resp, diagnostic)
		if !ok {
			state, reason = "failed", why
			// 明确异常长度继续沿bh.046关闭调度，模型不符/普通失败不直接改总调度。
			if diagnostic.DegradedSignal {
				final, e := s.gateway.accountRepo.GetByID(ctx, account.ID)
				if e == nil && codexTicketCollectionAllowed(ctx, final) && final.GetOpenAIAccessToken() == token && codexTicketKey(cfg, final, model, token) == key {
					diagnostic.Scheduling = s.applyTicketScheduling(ctx, cfg, final, false)
				}
			}
			return false, canRetry
		}
	}
	ticket := codexTicketValue{Attempts: attempt, State: extractOpenAICodexTurnState(resp.Header), ExpiresAt: time.Now().Add(cfg.ticketTTL())}
	if cfg.VerifiedFlow {
		ticket.Verified = true
		ticket.BusinessFingerprint = ticketBusinessFingerprint(fresh)
	}
	diagnostic.HTTPStatus = resp.StatusCode
	diagnostic.HeaderPresent = len(resp.Header.Values(openAICodexTurnStateHeader)) > 0
	diagnostic.HeaderLength = len(ticket.State)
	// 只接受原有安全格式的STATE；其它长度、HTTP错误或损坏头不产生调度结论。
	diagnostic.DegradedSignal = resp.StatusCode == http.StatusOK && cfg.DegradedSignalLength > 0 && validCodexTicket(ticket, cfg.DegradedSignalLength)
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
	if diagnostic.DegradedSignal {
		final, err := s.gateway.accountRepo.GetByID(ctx, account.ID)
		if err == nil && codexTicketCollectionAllowed(ctx, final) && final.IsModelSupported(model) && final.GetOpenAIAccessToken() == token && codexTicketKey(cfg, final, model, token) == key {
			diagnostic.Scheduling = s.applyTicketScheduling(ctx, cfg, final, false)
		} else {
			diagnostic.Scheduling = "stale"
		}
	}
	// 历史配置两长度相同时，明确异常优先，不能同轮先关再开。
	if resp.StatusCode != http.StatusOK || diagnostic.DegradedSignal || !validCodexTicket(ticket, cfg.targetLength()) {
		reason = "upstream"
		if resp.StatusCode == http.StatusOK {
			state, reason = "missing", "invalid_ticket"
		}
		readTicketFailureDiagnostic(resp.Body, diagnostic, cancel)
		if diagnostic.ErrorKind != "" {
			state, reason = "failed", "upstream"
		}
		diagnostic.RetryNotBefore = codexTicketRetryNotBefore(resp.Header.Get("Retry-After"), resp.StatusCode, diagnostic.ErrorKind)
		if diagnostic.RetryNotBefore == nil && (resp.StatusCode == 401 || resp.StatusCode == 403 || diagnostic.ErrorKind == "auth" || diagnostic.ErrorKind == "quota") {
			at := time.Now().Add(5 * time.Minute)
			diagnostic.RetryNotBefore = &at
		}
		if diagnostic.RetryNotBefore != nil {
			return false, false
		}
		// 授权/额度/明确限流不是通过切换 IP 无限重试可解决的问题。
		if resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 429 || diagnostic.ErrorKind == "auth" || diagnostic.ErrorKind == "quota" || diagnostic.ErrorKind == "rate_limit" || diagnostic.ErrorKind == "invalid_request" {
			return false, false
		}
		return false, ctx.Err() == nil
	}
	reason = "cancelled"
	if !s.ticketConfigCurrent(cfg) {
		return false, false
	}
	final, err := s.gateway.accountRepo.GetByID(ctx, account.ID)
	if err != nil || !codexTicketCollectionAllowed(ctx, final) || !final.IsModelSupported(model) || final.GetOpenAIAccessToken() != token || codexTicketKey(cfg, final, model, token) != key {
		return false, false
	}
	raw, _ := json.Marshal(ticket)
	reason = "storage"
	encrypted, err := s.cipher.Encrypt(string(raw))
	if err != nil {
		return false, false
	}
	if s.cache.Set(ctx, key, encrypted, cfg.ticketTTL()) != nil {
		return false, false
	}
	// 手动/自动成功都开调度，必须先存票；调度写失败不谎称成功，也不删除已保存票据。
	diagnostic.Scheduling = s.applyTicketScheduling(ctx, cfg, final, true)
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
