package httpapi

import (
	"context"
	"time"

	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/settings"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
	"github.com/TokenFlux/TokenRouter/internal/gateway/testkit"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
)

// responsesFixtureOptions 只描述 Responses 断言使用的请求、响应和图片桥接选项。
type responsesFixtureOptions struct {
	Request        OpenAIRequestOptions
	Response       OpenAIResponseOptions
	Headers        egress.ResponseHeaderOptions
	Health         provider.HealthOptions
	ForcedTemplate string
	ImageBridge    bool
}

type responsesFixtureInputs struct {
	providers       gatewayadapter.ExecutionProviderStore
	health          *provideradapter.UpstreamHealth
	headers         *egress.CompiledHeaderFilter
	profiles        *egressprovider.TLSProfiles
	routers         *egress.TLSFingerprintRouterService
	credentials     *provider.OpenAIExecutionCredentials
	registerTaskURL string
	grokTokens      *provider.GrokTokenSource
	readers         *gatewayadapter.RuntimeReaders
	cache           session.GatewayCache
	compactModel    string
	transport       httpclient.UpstreamTransport
	options         *responsesFixtureOptions
}

// newResponsesFixture 直接构造原生单次 HTTP 链，不创建提供商选择器、WS 池或完成队列。
func newResponsesFixture(v responsesFixtureInputs) *OpenAIResponsesExecutor {
	aux := newAuxiliaryFixture(auxiliaryFixtureInputs{transport: v.transport, store: v.providers, observer: v.health, profiles: v.profiles, credentials: v.credentials})
	options := responsesFixtureOptions{}
	if v.options != nil {
		options = *v.options
	}
	aux.Requests.Options = options.Request
	aux.Requests.Readers = v.readers
	aux.Requests.Routers = v.routers
	aux.Requests.ClientPolicy.Routers = v.routers
	aux.Requests.ClientPolicy.ForceCLI = options.Request.ForceCLI
	if v.readers != nil {
		aux.Requests.ClientPolicy.AllowClaudeCode = v.readers.Gateway.IsOpenAIAllowClaudeCodeCodexPluginEnabled
		aux.Requests.ClientPolicy.BrowserUserAgent = v.readers.Gateway.GetOpenAICodexUserAgent
		aux.Output.TTFT = v.readers.Gateway.GetOpenAITTFTMode
	}
	if v.registerTaskURL != "" {
		aux.Requests.Identity = gatewayadapter.NewExecutionAgentIdentity(&provider.OpenAITaskCoordinator{}, v.providers, func(ctx context.Context, value *provider.Record) (string, error) {
			return provideradapter.RegisterAgentIdentityTask(ctx, value, v.registerTaskURL)
		}, nil)
	}
	aux.Output.Headers = v.headers
	aux.Output.Observer = v.health
	aux.Output.ProxyCircuit = egress.NewProxyStreamCircuit(egress.DefaultProxyStreamCircuitSettings())
	aux.Output.Redact = aux.Requests.Identity.Redact
	aux.Output.Options = options.Response
	aux.Output.Options.Configured = v.options != nil
	aux.Output.Options.ResponseHeadersEnabled = options.Headers.Enabled
	if aux.Output.Options.ReadLimit == 0 {
		aux.Output.Options.ReadLimit = 128 * 1024 * 1024
	}
	aux.Output.Corrector = openai.NewCodexToolCorrector()
	aux.Output.Turns = aux.Requests.Turns
	history, _ := v.cache.(session.ReasoningContentCache)
	aux.Output.Reasoning = &session.ReasoningHistory{Cache: history}
	store := session.NewOpenAIWSStateStore(v.cache, gatewayadapter.LogOpenAIWSModeInfo)
	aux.Output.Responses = store
	aux.Output.ResponseTTL = func() time.Duration { return time.Hour }
	requests := aux.Requests
	credentials := testkit.RequestCredentials(v.providers, aux.Requests.Credentials, v.grokTokens, aux.Output.Health.Runtime)
	routes := gatewayadapter.GrokRoutes{Validate: options.Request.URLPolicy.Validate}
	requests.GrokRoutes = routes
	grok := &GrokExecutor{Credentials: credentials, Transport: v.transport, Output: aux.Output, Health: aux.Output.GrokHealth, Routes: routes, Failure: requests.Failure}
	text := &OpenAITextExecutor{ForcedTemplate: options.ForcedTemplate, Requests: requests, Output: aux.Output, Grok: grok, Credentials: credentials, FastPolicy: &gatewayadapter.ExecutionFastPolicy{Readers: v.readers}, Continuation: &session.CompatResponses{TTL: aux.Output.ResponseTTL}, PromptCache: session.NewAnthropicPromptCache(time.Now), CodexUsage: aux.CodexUsage, ResponseTTL: aux.Output.ResponseTTL, Compact: &CompactExecutor{Models: gatewayadapter.CompactModels{Default: v.compactModel}}}
	return &OpenAIResponsesExecutor{Requests: requests, Output: aux.Output, Text: text, Grok: grok, Lineage: &OpenAIEncryptedLineage{Store: store, TTL: aux.Output.ResponseTTL}, ImageBridge: &gatewayadapter.ResponseImagePolicy{DefaultEnabled: options.ImageBridge}, ResolveTransport: func(*gatewayadapter.ExecutionProvider) egress.OpenAIWSProtocolDecision {
		return egress.OpenAIWSProtocolDecision{Transport: egress.OpenAIUpstreamTransportHTTPSSE}
	}}
}

// compileHTTPFixtureHeaders 使用实际 Header 策略，保留 nil 配置语义。
func compileHTTPFixtureHeaders(options *responsesFixtureOptions) *egress.CompiledHeaderFilter {
	if options == nil {
		return nil
	}
	return egress.CompileHeaderFilter(options.Headers)
}

// newHTTPReadersFixture 只构造测试需要的动态读取端口。
func newHTTPReadersFixture(repo settings.Repository, _ *responsesFixtureOptions) *gatewayadapter.RuntimeReaders {
	if repo != nil {
		repo = settings.New(repo)
	}
	return testkit.RuntimeReaders(repo)
}

// newHTTPHealthFixture 把测试预算传给唯一健康实现。
func newHTTPHealthFixture(store gatewayadapter.ExecutionProviderStore, options *responsesFixtureOptions, cache provider.TempUnschedCache, health provider.HealthOptions, readers *gatewayadapter.RuntimeReaders) *provideradapter.UpstreamHealth {
	if options != nil {
		health.UnauthorizedCooldownMinutes = options.Health.UnauthorizedCooldownMinutes
		health.OverloadMinutes = options.Health.OverloadMinutes
		health.CNIntervalMinutes = options.Health.CNIntervalMinutes
	}
	return testkit.NewHealthObserver(testkit.HealthInput{Store: store, Cache: cache, Options: health, Readers: readers})
}

// httpFixtureRuntimeBlocked 只投影身份，运行阻断仍由提供商模块判断。
// httpFixtureRuntimeBlocked 只投影凭据身份，停调与恢复仍由提供商运行状态判断。
func httpFixtureRuntimeBlocked(s *OpenAIResponsesExecutor, target *gatewayadapter.ExecutionProvider) bool {
	return s.Output.Health.Runtime.Blocked(target.Record.ID, func() string {
		return provider.RefreshCredentialIdentity(target.View())
	})
}
