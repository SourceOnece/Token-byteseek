package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/gateway/requeststate"
	"github.com/TokenFlux/TokenRouter/internal/protocol"
	protocolopenai "github.com/TokenFlux/TokenRouter/internal/protocol/openai"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/upstream"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	gatewayws "github.com/TokenFlux/TokenRouter/internal/gateway/ws"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// executeWSIngressAdapter 处理客户端入站 WebSocket（OpenAI Responses WS Mode）并转发到上游。
// 当前实现按“单请求 -> 终止事件 -> 下一请求”的顺序代理，适配 Codex CLI 的 turn 模式。
func (s *OpenAIWebSocketExecutor) executeWSIngressAdapter(
	ctx context.Context,
	c *gin.Context,
	clientConn *coderws.Conn,
	provider *gatewayprovider.ExecutionProvider,
	token string,
	firstClientMessage []byte,
	hooks *gatewayws.OpenAIIngressHooks,
) (returnErr error) {
	if s == nil {
		return errors.New("service is nil")
	}
	if c == nil {
		return errors.New("gin context is nil")
	}
	if clientConn == nil {
		return errors.New("client websocket is nil")
	}
	if provider == nil {
		return errors.New("provider is nil")
	}
	// 复用 Gin 上下文时清理上一个提供商留下的工具名称映射。
	SetCodexToolNameReverse(c, nil)
	if _, err := PrepareCodexIdentity(ctx, c, s.Requests.Providers, provider); err != nil {
		return err
	}
	if err := validateOpenAIWSBearerToken(provider, token); err != nil {
		return err
	}

	tlsRouterMatch := s.Requests.MatchTLS(c, provider)

	// 预取一次 OpenAI Fast Policy settings，绑定到 ctx，让该 WS session
	// 内所有帧的 evaluateOpenAIFastPolicy 调用复用同一份快照，避免每帧
	// 进入 DB / settingRepo。Trade-off 见 gatewayprovider.WithFastPolicyContext 注释。
	if s.Requests.Readers != nil {
		if settings, err := s.Requests.Readers.Gateway.GetOpenAIFastPolicySettings(ctx); err == nil && settings != nil {
			ctx = gatewayprovider.WithFastPolicyContext(ctx, settings)
		}
	}

	// The handler normally owns this registration across retry attempts. Direct
	// callers still get the same session-scoped preemption behavior here.
	if preemptCtx, cleanupPreempt, armed := s.BeginOpenAIWSIngressSessionPreemptionWithClient(ctx, c, provider, firstClientMessage, clientConn); armed {
		ctx = preemptCtx
		defer cleanupPreempt()
		defer func() {
			if isOpenAIWSSessionPreempted(ctx) {
				returnErr = gatewayws.ErrSessionPreempted
			}
		}()
	}

	var routeErr error
	provider, routeErr = gatewayprovider.ProviderForProtocolAttempt(ctx, provider)
	if routeErr != nil {
		return routeErr
	}
	wsDecision := s.Selection.ResolveTransport(provider)
	forceHTTPBridge := provider.Record.Platform == capability.PlatformGrok || provider.Route.Protocol() == protocol.ProtocolOpenAIResponses
	modeRouterV2Enabled := s != nil && s.Options != nil && s.Options.ModeRouterV2Enabled
	ingressMode := providercore.OpenAIWSIngressModeCtxPool
	if modeRouterV2Enabled && !forceHTTPBridge {
		ingressMode = provider.View().ResolveOpenAIResponsesWebSocketV2Mode(s.Options.IngressModeDefault)
		if ingressMode == providercore.OpenAIWSIngressModeOff {
			return NewOpenAIWSClientCloseError(
				coderws.StatusPolicyViolation,
				"websocket mode is disabled for this provider",
				nil,
			)
		}
		switch ingressMode {
		case providercore.OpenAIWSIngressModePassthrough:
			if wsDecision.Transport != egress.OpenAIUpstreamTransportResponsesWebsocketV2 {
				return fmt.Errorf("websocket ingress requires ws_v2 transport, got=%s", wsDecision.Transport)
			}
			if s.shouldBridgeOpenAIWSPassthroughFirstMessage(provider, firstClientMessage) {
				forceHTTPBridge = true
				break
			}
			// 透传 relay 通过 TurnStarted 记录每个 turn 的开始时刻，但不触发
			// BeforeTurn；因此仍只有建连时的利润准入门，没有 turn 级复核。
			// handler 计费在 turn 定价未冻结时回退到对应的 turn 开始时刻。
			return s.proxyResponsesWebSocketV2Passthrough(
				ctx,
				c,
				clientConn,
				provider,
				token,
				firstClientMessage,
				hooks,
				wsDecision,
				tlsRouterMatch,
			)
		case providercore.OpenAIWSIngressModeHTTPBridge:
			forceHTTPBridge = true
		case providercore.OpenAIWSIngressModeCtxPool, providercore.OpenAIWSIngressModeShared, providercore.OpenAIWSIngressModeDedicated:
			// continue
		default:
			return NewOpenAIWSClientCloseError(
				coderws.StatusPolicyViolation,
				"websocket mode only supports ctx_pool/passthrough/http_bridge",
				nil,
			)
		}
	}
	if !forceHTTPBridge && wsDecision.Transport != egress.OpenAIUpstreamTransportResponsesWebsocketV2 {
		return fmt.Errorf("websocket ingress requires ws_v2 transport, got=%s", wsDecision.Transport)
	}
	dedicatedMode := modeRouterV2Enabled && ingressMode == providercore.OpenAIWSIngressModeDedicated

	wsURL := ""
	wsHost := "-"
	wsPath := "-"
	if forceHTTPBridge {
		wsHost = "xai-http-bridge"
		wsPath = "/v1/responses"
	} else {
		var err error
		wsURL, err = s.buildOpenAIResponsesWSURL(provider)
		if err != nil {
			return fmt.Errorf("build ws url: %w", err)
		}
		if parsedURL, parseErr := url.Parse(wsURL); parseErr == nil && parsedURL != nil {
			wsHost = gatewayprovider.NormalizeOpenAIWSLogValue(parsedURL.Host)
			wsPath = gatewayprovider.NormalizeOpenAIWSLogValue(parsedURL.Path)
		}
	}
	debugEnabled := gatewayprovider.IsOpenAIWSModeDebugEnabled()
	isCodexCLI := openai.IsCodexOfficialClientByHeaders(c.GetHeader("User-Agent"), c.GetHeader("originator")) || (s.Options != nil && s.Requests.Options.ForceCLI)

	state := &gatewayws.IngressState{TurnState: strings.TrimSpace(c.GetHeader(openAIWSTurnStateHeader))}

	normalizer := &gatewayws.RequestNormalizer{State: state, Options: gatewayws.NormalizeOptions{ProviderID: provider.Record.ID, OAuth: provider.View().IsOpenAIOAuth(), ForceHTTPBridge: forceHTTPBridge}, Port: &wsRequestAdapter{wsPassthroughAdapter: &wsPassthroughAdapter{service: s, request: c, provider: provider, hooks: hooks}, client: clientConn, isCodex: isCodexCLI}}
	parseClientPayload := func(raw []byte, replace bool, turn int) (gatewayws.ClientPayload, error) {
		return normalizer.Normalize(ctx, raw, replace, turn)
	}

	writeClientMessage := func(message []byte) error {
		writeCtx, cancel := newOpenAIWSDownstreamWriteContext(ctx, hooks, s.openAIWSWriteTimeout())
		defer cancel()
		message = RestoreCodexToolNamesFromContext(c, message)
		return clientConn.Write(writeCtx, coderws.MessageText, message)
	}

	readClientMessage := func() ([]byte, error) {
		idleTimeout := s.openAIWSIngressInterTurnIdleTimeout()
		msgType, payload, readErr := ReadOpenAIWSClientMessage(
			ctx,
			clientConn,
			idleTimeout,
			coderws.StatusNormalClosure,
			"websocket idle timeout",
		)
		if readErr != nil {
			var closeErr *OpenAIWSClientCloseError
			if errors.As(readErr, &closeErr) && closeErr.StatusCode() == coderws.StatusNormalClosure {
				gatewayprovider.LogOpenAIWSModeInfo("ingress_ws_inter_turn_idle_timeout provider_id=%d timeout_seconds=%d", provider.Record.ID, int(idleTimeout.Seconds()))
			}
			return nil, readErr
		}
		if msgType != coderws.MessageText && msgType != coderws.MessageBinary {
			return nil, NewOpenAIWSClientCloseError(
				coderws.StatusPolicyViolation,
				fmt.Sprintf("unsupported websocket client message type: %s", msgType.String()),
				nil,
			)
		}
		return payload, nil
	}

	groupID := OpenAIResponseGroupID(c)
	stateStore := s.State
	var baseAcquireReq openai.WSAcquireRequest
	var pool *openai.WSConnPool
	openPool := func(firstPayload gatewayws.ClientPayload) error {
		firstRoutingFields := gjson.GetManyBytes(firstPayload.PayloadRaw, "model", "service_tier")
		wsHeaders, sessionResolution, buildHdrErr := s.buildOpenAIWSHeaders(
			ctx,
			c,
			provider,
			token,
			wsDecision,
			isCodexCLI,
			state.TurnState,
			strings.TrimSpace(c.GetHeader(openai.WSTurnMetadataHeader)),
			firstPayload.PromptCacheKey,
			firstRoutingFields[0].String(),
			firstRoutingFields[1].String(),
			tlsRouterMatch,
		)
		if buildHdrErr != nil {
			return fmt.Errorf("build ws headers: %w", buildHdrErr)
		}
		tlsProfile, tlsProfileKey := s.Requests.WSTLSProfile(provider, tlsRouterMatch)
		baseAcquireReq = openai.WSAcquireRequest{
			Observer: sessionResolution.Observer,
			Provider: openAIWSPoolProviderView(provider),
			WSURL:    wsURL,
			Headers:  wsHeaders,
			HeadersFactory: func(factoryCtx context.Context, headers http.Header) (http.Header, error) {
				return s.Requests.Identity.RefreshHeaders(factoryCtx, provider, headers)
			},
			TLSProfile:    tlsProfile,
			TLSProfileKey: tlsProfileKey,
			ProxyURL: func() string {
				if provider.Record.ProxyID != nil && provider.Record.Proxy != nil {
					return provider.Record.Proxy.URL()
				}
				return ""
			}(),
			ForceNewConn: false,
		}
		pool = s.Connections.Pool()
		if pool == nil {
			return errors.New("openai ws conn pool is nil")
		}
		gatewayprovider.LogOpenAIWSModeInfo(
			"ingress_ws_protocol_confirm provider_id=%d provider_type=%s transport=%s ws_host=%s ws_path=%s ws_mode=%s store_disabled=%v has_session_hash=%v has_previous_response_id=%v",
			provider.Record.ID,
			provider.Record.Type, gatewayprovider.NormalizeOpenAIWSLogValue(string(wsDecision.Transport)), wsHost,
			wsPath, gatewayprovider.NormalizeOpenAIWSLogValue(ingressMode), state.StoreDisabled,
			state.SessionHash != "",
			firstPayload.PreviousResponseID != "",
		)

		if debugEnabled {
			gatewayprovider.LogOpenAIWSModeDebug(
				"ingress_ws_start provider_id=%d provider_type=%s transport=%s ws_host=%s preferred_conn_id=%s has_session_hash=%v has_previous_response_id=%v store_disabled=%v",
				provider.Record.ID,
				provider.Record.Type, gatewayprovider.NormalizeOpenAIWSLogValue(string(wsDecision.Transport)), wsHost, gatewayprovider.TruncateOpenAIWSLogValue(state.PreferredConnID, gatewayprovider.OpenAIWSIDValueMaxLen), state.SessionHash != "",
				firstPayload.PreviousResponseID != "",
				state.StoreDisabled,
			)
		}
		if firstPayload.PreviousResponseID != "" {
			firstPreviousResponseIDKind := protocolopenai.ClassifyOpenAIPreviousResponseIDKind(firstPayload.PreviousResponseID)
			gatewayprovider.LogOpenAIWSModeInfo(
				"ingress_ws_continuation_probe provider_id=%d turn=%d previous_response_id=%s previous_response_id_kind=%s preferred_conn_id=%s session_hash=%s header_session_id=%s header_conversation_id=%s has_turn_state=%v turn_state_len=%d has_prompt_cache_key=%v store_disabled=%v",
				provider.Record.ID,
				1, gatewayprovider.TruncateOpenAIWSLogValue(firstPayload.PreviousResponseID, gatewayprovider.OpenAIWSIDValueMaxLen), gatewayprovider.NormalizeOpenAIWSLogValue(firstPreviousResponseIDKind), gatewayprovider.TruncateOpenAIWSLogValue(state.PreferredConnID, gatewayprovider.OpenAIWSIDValueMaxLen), gatewayprovider.TruncateOpenAIWSLogValue(state.SessionHash, 12), gatewayprovider.OpenAIWSHeaderValueForLog(baseAcquireReq.Headers, "session_id"), gatewayprovider.OpenAIWSHeaderValueForLog(baseAcquireReq.Headers, "conversation_id"), state.TurnState != "",
				len(state.TurnState),
				firstPayload.PromptCacheKey != "",
				state.StoreDisabled,
			)
		}

		return nil
	}

	acquireTimeout := s.openAIWSAcquireTimeout()
	if acquireTimeout <= 0 {
		acquireTimeout = 30 * time.Second
	}

	acquireTurnLease := func(turn int, preferred string, forcePreferredConn bool, allowRecovery bool) (*openai.WSConnLease, error) {
		req := openai.CloneWSAcquireRequest(baseAcquireReq)
		req.PreferredConnID = strings.TrimSpace(preferred)
		req.ForcePreferredConn = forcePreferredConn
		// dedicated 模式下每次获取均新建连接，避免跨会话复用残留上下文。
		req.ForceNewConn = dedicatedMode
		acquireCtx, acquireCancel := context.WithTimeout(ctx, acquireTimeout)
		lease, acquireErr := pool.Acquire(acquireCtx, req)
		acquireCancel()
		var dialErr *openai.WSDialError
		if acquireErr != nil && s.Requests.Identity.UsesAgentIdentity(ctx, provider) && errors.As(acquireErr, &dialErr) && openai.IsAgentTaskInvalidWSDialError(dialErr) && allowRecovery {
			return nil, &gatewayws.AcquireRecoveryError{Err: acquireErr}
		}
		if acquireErr != nil {
			canonicalModel := gatewayprovider.ExecutionModelPolicy(provider).CanonicalSchedulingModel(state.OriginalModel)
			errorDecision := s.handleOpenAIWSDialTransientFailure(ctx, provider, canonicalModel, acquireErr)
			dialStatus, dialClass, dialCloseStatus, dialCloseReason, dialRespServer, dialRespVia, dialRespCFRay, dialRespReqID := gatewayprovider.SummarizeOpenAIWSDialError(acquireErr)
			gatewayprovider.LogOpenAIWSModeInfo(
				"ingress_ws_upstream_acquire_fail provider_id=%d turn=%d reason=%s dial_status=%d dial_class=%s dial_close_status=%s dial_close_reason=%s dial_resp_server=%s dial_resp_via=%s dial_resp_cf_ray=%s dial_resp_x_request_id=%s cause=%s preferred_conn_id=%s force_preferred_conn=%v ws_host=%s ws_path=%s proxy_enabled=%v",
				provider.Record.ID,
				turn, gatewayprovider.NormalizeOpenAIWSLogValue(openai.ClassifyWSAcquireError(acquireErr)), dialStatus,
				dialClass,
				dialCloseStatus, gatewayprovider.TruncateOpenAIWSLogValue(dialCloseReason, gatewayprovider.OpenAIWSHeaderValueMaxLen), dialRespServer,
				dialRespVia,
				dialRespCFRay,
				dialRespReqID, gatewayprovider.TruncateOpenAIWSLogValue(acquireErr.Error(), gatewayprovider.OpenAIWSLogValueMaxLen), gatewayprovider.TruncateOpenAIWSLogValue(preferred, gatewayprovider.OpenAIWSIDValueMaxLen), forcePreferredConn,
				wsHost,
				wsPath,
				provider.Record.ProxyID != nil && provider.Record.Proxy != nil,
			)
			var dialErr *openai.WSDialError
			if errors.As(acquireErr, &dialErr) && dialErr != nil && dialErr.StatusCode != 0 {
				if turn == 1 && errorDecision.ShouldReturnGenericError() {
					return nil, openAIWSGenericPolicyCloseError(dialErr.StatusCode)
				}
				if turn == 1 && errorDecision.ShouldFailoverWithDefaults(
					gatewayprovider.ExecutionErrorPolicy(provider),
					dialErr.StatusCode,
					dialErr.StatusCode == http.StatusTooManyRequests,
					s.shouldFailoverOpenAIWSError(provider, dialErr.StatusCode, dialErr.ResponseBody),
				) {
					return nil, gatewayprovider.NewOpenAIUpstreamFailure(
						dialErr.StatusCode,
						dialErr.ResponseHeaders,
						dialErr.ResponseBody,
						upstream.ExtractErrorMessage(dialErr.ResponseBody),
						errorDecision.RetryableOnSameProvider(gatewayprovider.ExecutionErrorPolicy(provider), dialErr.StatusCode),
					)
				}
			}
			if errors.Is(acquireErr, openai.ErrOpenAIWSPreferredConnUnavailable) {
				return nil, NewOpenAIWSClientCloseError(
					coderws.StatusPolicyViolation,
					"upstream continuation connection is unavailable; please restart the conversation",
					acquireErr,
				)
			}
			if errors.Is(acquireErr, context.DeadlineExceeded) || errors.Is(acquireErr, openai.ErrOpenAIWSConnQueueFull) {
				return nil, NewOpenAIWSClientCloseError(
					coderws.StatusTryAgainLater,
					"upstream websocket is busy, please retry later",
					acquireErr,
				)
			}
			return nil, acquireErr
		}
		connID := strings.TrimSpace(lease.ConnID())
		if handshakeTurnState := strings.TrimSpace(lease.HandshakeHeader(openAIWSTurnStateHeader)); handshakeTurnState != "" {
			state.TurnState = handshakeTurnState
			if stateStore != nil && state.SessionHash != "" {
				stateStore.BindSessionTurnState(groupID, state.SessionHash, handshakeTurnState, s.Selection.SessionStickyTTL())
			}
			updatedHeaders := upstream.CloneHeader(baseAcquireReq.Headers)
			if updatedHeaders == nil {
				updatedHeaders = make(http.Header)
			}
			updatedHeaders.Set(openAIWSTurnStateHeader, handshakeTurnState)
			baseAcquireReq.Headers = updatedHeaders
		}
		gatewayprovider.LogOpenAIWSModeInfo(
			"ingress_ws_upstream_connected provider_id=%d turn=%d conn_id=%s conn_reused=%v conn_pick_ms=%d queue_wait_ms=%d preferred_conn_id=%s",
			provider.Record.ID,
			turn, gatewayprovider.TruncateOpenAIWSLogValue(connID, gatewayprovider.OpenAIWSIDValueMaxLen), lease.Reused(),
			lease.ConnPickDuration().Milliseconds(),
			lease.QueueWaitDuration().Milliseconds(), gatewayprovider.TruncateOpenAIWSLogValue(preferred, gatewayprovider.OpenAIWSIDValueMaxLen),
		)
		return lease, nil
	}

	port := &wsIngressAdapter{
		ResetBridgeWindowFn: func() { SetOpenAIWSHTTPBridgeToolState(c, requeststate.WSBridgeTools{}) },
		ParseFn:             parseClientPayload, ReadFn: readClientMessage,
		GenerateHashFn:  func(body []byte) string { return GenerateOpenAISessionHash(c, body) },
		StoreDisabledFn: func(body []byte) bool { return s.isOpenAIWSStoreDisabledInRequestRaw(body, provider) },
		ShouldBridgeFn: func(payload gatewayws.ClientPayload) bool {
			return forceHTTPBridge || s.shouldBridgeOpenAIWSHTTP(provider, payload.PayloadBytes, payload.PreviousResponseID)
		},
		InvalidFn: s.Lineage.Digests, StripFn: s.Lineage.Strip,
		BridgeIdentityFn: func(body []byte, model string) (string, error) {
			return resolveGrokWSCacheIdentity(c, provider, body, model)
		},
		BridgeFn: func(ctx context.Context, input gatewayws.ClientPayload, body []byte, identity string, turn int) (*gatewayws.ForwardResult, error) {
			result, err := s.proxyOpenAIWSHTTPBridgeTurn(ctx, c, provider, token, body, len(body), input.OriginalModel, input.RoutingModel, input.ImageBillingModel, input.ImageSizeTier, input.ImageInputSize, identity, turn, writeClientMessage, tlsRouterMatch)
			return gatewayprovider.ProjectWSResult(result), err
		},
		SetStateFn: func(turnState, sessionHash string) {
			if turnState != "" && c != nil && c.Request != nil {
				c.Request.Header.Set(openAIWSTurnStateHeader, turnState)
			}
			if c != nil && sessionHash != "" {
				c.Set(OpenAIWSIngressLineageContextKey, sessionHash)
			}
		},
		OpenPoolFn: openPool,
		RecoverAcquireFn: func(ctx context.Context) error {
			return s.Requests.Identity.Recover(ctx, provider, provider.View().GetCredential("task_id"))
		},
		AcquireFn: func(turn int, preferred string, force bool, allowRecovery bool) (gatewayws.ConnLease, error) {
			lease, err := acquireTurnLease(turn, preferred, force, allowRecovery)
			if err != nil || lease == nil {
				return nil, err
			}
			return &wsIngressLease{lease: lease}, nil
		},
		RelayFn: func(turn int, lease gatewayws.ConnLease, input gatewayws.ClientPayload) (*gatewayws.ForwardResult, error) {
			streamPort := &wsStreamAdapter{wsPassthroughAdapter: &wsPassthroughAdapter{service: s, request: c, provider: provider, hooks: hooks}, state: state, groupID: groupID, writeClient: writeClientMessage}
			var streamHooks *gatewayws.StreamHooks
			if hooks != nil {
				streamHooks = &gatewayws.StreamHooks{OnUpstreamError: hooks.OnUpstreamError}
			}
			// 此连接句柄只能来自本适配器；保持错误类型按编程错误失败。
			typedLease, ok := lease.(*wsIngressLease)
			if !ok {
				panic("unexpected websocket ingress lease type")
			}
			return gatewayws.RelayTurn(ctx, streamPort, typedLease, input, turn, gatewayws.StreamOptions{ProviderID: provider.Record.ID, WriteTimeout: s.openAIWSWriteTimeout(), ReadTimeout: s.openAIWSReadTimeout(), PreviousRecovery: s.openAIWSIngressPreviousResponseRecoveryEnabled(), Debug: debugEnabled}, streamHooks)
		},

		PinFn:       func(id int64, conn string) bool { return pool.PinConn(id, conn) },
		UnpinFn:     func(id int64, conn string) { pool.UnpinConn(id, conn) },
		HeaderFn:    func(key string) string { return gatewayprovider.OpenAIWSHeaderValueForLog(baseAcquireReq.Headers, key) },
		BindOwnerFn: func(ctx context.Context, responseID string) { s.bindOpenAIWSResponseSessionOwner(ctx, c, responseID) },
		UpdateHeadersFn: func(nextPayload gatewayws.ClientPayload, turnState string) {
			nextRoutingFields := gjson.GetManyBytes(nextPayload.PayloadRaw, "model", "service_tier")
			if nextPayload.PromptCacheKey != "" {
				updatedHeaders, resolution, err := s.buildOpenAIWSHeaders(ctx, c, provider, token, wsDecision, isCodexCLI, turnState, strings.TrimSpace(c.GetHeader(openai.WSTurnMetadataHeader)), nextPayload.PromptCacheKey, nextRoutingFields[0].String(), nextRoutingFields[1].String(), tlsRouterMatch)
				if err != nil {
					gatewayprovider.LogOpenAIWSModeInfo("ingress_ws_update_headers_failed provider_id=%d err=%v", provider.Record.ID, err)
				} else {
					baseAcquireReq.Headers = updatedHeaders
					baseAcquireReq.Observer = resolution.Observer
				}
			}
			SetOpenAICodexRoutingHint(baseAcquireReq.Headers, provider, nextRoutingFields[0].String(), nextRoutingFields[1].String())
		},
	}
	runtime := gatewayws.IngressSession{State: state, Store: stateStore, Codec: wsReplayCodec{}, Port: port, Hooks: wsIngressHooks(hooks), Options: gatewayws.IngressOptions{
		ProviderID: provider.Record.ID, ProviderType: provider.Record.Type, Platform: provider.Record.Platform, GroupID: groupID,
		Debug: debugEnabled, BridgeThreshold: s.openAIWSHTTPBridgeThresholdBytes(), PreviousRecovery: s.openAIWSIngressPreviousResponseRecoveryEnabled(), StoreDisabledMode: s.openAIWSStoreDisabledConnMode(),
		PreflightPingIdle: openAIWSIngressPreflightPingIdle, HealthCheckTimeout: openai.WSConnHealthCheckTimeout, ResponseStickyTTL: s.OpenAIHTTPResponseStickyTTL(), SessionStickyTTL: s.Selection.SessionStickyTTL(),
	}}
	return runtime.Run(ctx, firstClientMessage)
}
