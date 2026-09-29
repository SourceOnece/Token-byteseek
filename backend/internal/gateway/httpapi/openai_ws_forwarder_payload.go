package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	forward "github.com/TokenFlux/TokenRouter/internal/gateway/provider/openaiforward"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	upstreamopenai "github.com/TokenFlux/TokenRouter/internal/upstream/openai"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func validateOpenAIWSBearerToken(provider *gatewayprovider.ExecutionProvider, token string) error {
	if provider == nil {
		return errors.New("provider is nil")
	}
	if strings.TrimSpace(token) == "" && !provider.View().IsOpenAIAgentIdentity() {
		return errors.New("token is empty")
	}
	return nil
}

func (s *OpenAIWebSocketExecutor) buildOpenAIResponsesWSURL(provider *gatewayprovider.ExecutionProvider) (string, error) {
	if provider == nil {
		return "", errors.New("provider is nil")
	}
	var targetURL string
	switch provider.Record.Type {
	case capability.ProviderTypeOAuth:
		targetURL = chatgptCodexURL
	case capability.ProviderTypeSetupToken:
		if provider.View().IsOpenAIOAuthLike() {
			targetURL = chatgptCodexURL
		} else {
			targetURL = openaiPlatformAPIURL
		}
	case capability.ProviderTypeAPIKey:
		baseURL := gatewayprovider.ExecutionProtocolTarget(provider).GetOpenAIBaseURL()
		if _, unified := provider.Record.Credentials[providercore.UpstreamProtocolsKey]; gatewayprovider.ExecutionProtocolTarget(provider).UsesNativeCNResponses() && (unified || gatewayprovider.ExecutionProtocolTarget(provider).IsAdaptiveAPIProtocol()) {
			baseURL = gatewayprovider.ExecutionProtocolTarget(provider).GetCNProtocolBaseURL(providercore.APIProtocolResponses)
		}
		if baseURL == "" {
			targetURL = openaiPlatformAPIURL
		} else {
			validatedURL, err := s.Requests.ValidateBaseURL(baseURL)
			if err != nil {
				return "", err
			}
			targetURL = forward.ResponsesEndpoint(provider.Record.Platform, validatedURL)
		}
	default:
		targetURL = openaiPlatformAPIURL
	}

	parsed, err := url.Parse(strings.TrimSpace(targetURL))
	if err != nil {
		return "", fmt.Errorf("invalid target url: %w", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	case "wss", "ws":
		// 保持不变
	default:
		return "", fmt.Errorf("unsupported scheme for ws: %s", parsed.Scheme)
	}
	return parsed.String(), nil
}

func (s *OpenAIWebSocketExecutor) buildOpenAIWSHeaders(
	ctx context.Context,
	c *gin.Context,
	provider *gatewayprovider.ExecutionProvider,
	token string,
	decision egress.OpenAIWSProtocolDecision,
	isCodexCLI bool,
	turnState string,
	turnMetadata string,
	promptCacheKey string,
	routingModel string,
	routingServiceTier string,
	routerMatch ...egress.TLSFingerprintRouterMatchResult,
) (http.Header, OpenAIWSSessionHeaderResolution, error) {
	var sessionResolution OpenAIWSSessionHeaderResolution
	headers, err := upstreamopenai.BuildWSHeaders(ctx, upstreamopenai.WSHeaderOptions{
		AgentIdentity: provider != nil && provider.View().IsOpenAIAgentIdentity(), Token: token,
		TurnState: turnState, TurnMetadata: turnMetadata,
		BetaV1: openAIWSBetaV1Value, BetaV2: openAIWSBetaV2Value,
		LegacyWS: decision.Transport == egress.OpenAIUpstreamTransportResponsesWebsocket,
		ResolveSession: func() (string, string) {
			sessionResolution = ResolveOpenAIWSSessionHeaders(c, promptCacheKey)
			return sessionResolution.SessionID, sessionResolution.ConversationID
		},
		InboundHeaders: func() http.Header {
			if c != nil && c.Request != nil {
				return c.Request.Header
			}
			return nil
		},
		UserAgent: func() string {
			if c != nil {
				return c.GetHeader("User-Agent")
			}
			return ""
		},
		ApplyWSUserAgent: func(headers http.Header) {
			reqCtx := context.Background()
			if c != nil && c.Request != nil {
				reqCtx = c.Request.Context()
			}
			s.Requests.ApplyUserAgentHeader(reqCtx, c, provider, headers, true, routerMatch...)
		},
		ResponsesRequestOptions: upstreamopenai.ResponsesRequestOptions{
			UsesCodex: func() bool { return provider != nil && provider.View().UsesOpenAICodexProtocol() },
			APIKeyID:  func() int64 { return APIKeyIDFromContext(c) },
			IsolateSession: func(keyID int64, value string) string {
				return upstreamopenai.IsolateOpenAIUpstreamSessionID(keyID, provideradapter.CodexIdentityNamespace(CodexIdentityRecord(c, provider.View())), value)
			},
			ApplyProviderIdentity: func(headers http.Header) {
				upstreamopenai.ApplyCodexProviderIdentityHeaders(headers, provideradapter.CodexIdentityNamespace(CodexIdentityRecord(c, provider.View())), APIKeyIDFromContext(c))
			},
			ApplyFingerprint: func(headers http.Header) { ApplyStagedCodexFingerprintHeaders(c, provider.View(), headers) },
			ProviderHeaders: func(ctx context.Context, headers http.Header) error {
				return gatewayprovider.CredentialChatGPTHeaders(ctx, s.Requests.Providers, headers, provider)
			},
			Originator:      func() string { return ResolveOpenAIUpstreamOriginator(c, isCodexCLI, routerMatch...) },
			OverrideHeaders: gatewayprovider.BindExecutionHeaders(provider),
			BetaFeatures: func(headers http.Header) {
				ApplyOpenAICodexBetaFeatures(c, provider != nil && provider.View().IsOpenAIOAuthLike(), headers)
			},
			RoutingHint: func(headers http.Header, _ []byte) {
				SetOpenAICodexRoutingHint(headers, provider, routingModel, routingServiceTier)
			},
			Diagnostics: func(headers http.Header, _ []byte) {
				LogOpenAIRoutingDiagnostics(ctx, provider, string(decision.Transport), routingModel, routingServiceTier, strings.TrimSpace(headers.Get(OpenAICodexRoutingHintHeader)) != "", "soft_routing_hint")
			},
		},
	})
	if err == nil && s.Requests.Tickets != nil && provider != nil {
		sessionResolution.Observer, err = s.Requests.Tickets.ApplyWebSocket(ctx, provider.View(), routingModel, headers)
	}
	return headers, sessionResolution, err
}

func (s *OpenAIWebSocketExecutor) buildOpenAIWSCreatePayload(reqBody map[string]any, provider *gatewayprovider.ExecutionProvider) map[string]any {
	// OpenAI WS Mode 协议：response.create 字段与 HTTP /responses 基本一致。
	// 保留 stream 字段（与 Codex CLI 一致），仅移除 background。
	payload := make(map[string]any, len(reqBody)+1)
	for k, v := range reqBody {
		payload[k] = v
	}

	delete(payload, "background")
	if _, exists := payload["stream"]; !exists {
		payload["stream"] = true
	}
	payload["type"] = "response.create"

	// OAuth 默认保持 store=false，避免误依赖服务端历史。
	if provider != nil && provider.View().UsesOpenAICodexProtocol() && !s.isOpenAIWSStoreRecoveryAllowed(provider) {
		payload["store"] = false
	}
	return payload
}

func (s *OpenAIWebSocketExecutor) isOpenAIWSStoreRecoveryAllowed(provider *gatewayprovider.ExecutionProvider) bool {
	if provider != nil && provider.View().IsOpenAIWSAllowStoreRecoveryEnabled() {
		return true
	}
	if s != nil && s.Options != nil && s.Options.AllowStoreRecovery {
		return true
	}
	return false
}

func (s *OpenAIWebSocketExecutor) isOpenAIWSStoreDisabledInRequest(reqBody map[string]any, provider *gatewayprovider.ExecutionProvider) bool {
	if provider != nil && provider.View().UsesOpenAICodexProtocol() && !s.isOpenAIWSStoreRecoveryAllowed(provider) {
		return true
	}
	if len(reqBody) == 0 {
		return false
	}
	rawStore, ok := reqBody["store"]
	if !ok {
		return false
	}
	storeEnabled, ok := rawStore.(bool)
	if !ok {
		return false
	}
	return !storeEnabled
}

func (s *OpenAIWebSocketExecutor) isOpenAIWSStoreDisabledInRequestRaw(reqBody []byte, provider *gatewayprovider.ExecutionProvider) bool {
	if provider != nil && provider.View().UsesOpenAICodexProtocol() && !s.isOpenAIWSStoreRecoveryAllowed(provider) {
		return true
	}
	if len(reqBody) == 0 {
		return false
	}
	storeValue := gjson.GetBytes(reqBody, "store")
	if !storeValue.Exists() {
		return false
	}
	if storeValue.Type != gjson.True && storeValue.Type != gjson.False {
		return false
	}
	return !storeValue.Bool()
}

func (s *OpenAIWebSocketExecutor) openAIWSStoreDisabledConnMode() string {
	if s == nil || s.Options == nil {
		return openAIWSStoreDisabledConnModeStrict
	}
	mode := strings.ToLower(strings.TrimSpace(s.Options.StoreDisabledConnMode))
	switch mode {
	case openAIWSStoreDisabledConnModeStrict, openAIWSStoreDisabledConnModeAdaptive, openAIWSStoreDisabledConnModeOff:
		return mode
	case "":
		// 兼容旧配置：仅配置了布尔开关时按旧语义推导。
		if s.Options.StoreDisabledForceNewConn {
			return openAIWSStoreDisabledConnModeStrict
		}
		return openAIWSStoreDisabledConnModeOff
	default:
		return openAIWSStoreDisabledConnModeStrict
	}
}

// Replay 状态所有权不变式：replay 序列中的 json.RawMessage 正文一经放入即视为
// 不可变，所有持有者共享同一份字节，任何修改都必须整体替换元素或重建 payload。
// 序列头数组在跨持有者保存时必须新建（combineOpenAIWSReplayItems），禁止通过
// 共享头 append，否则会写入其他持有者可见的底层数组。
