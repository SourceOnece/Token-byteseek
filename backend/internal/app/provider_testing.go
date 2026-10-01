package app

import (
	"context"
	"log"
	"slices"
	"time"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway"
	openaiprotocol "github.com/TokenFlux/TokenRouter/internal/protocol/openai"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerhttp "github.com/TokenFlux/TokenRouter/internal/provider/httpapi"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/antigravity"
	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
)

// provideProviderTests 直接绑定原生提供商和平台目标，后台与 HTTP 共用同一入口。
func provideProviderTests(store *providerpostgres.ProviderStore, geminiToken *provider.GeminiTokenSource, claudeToken *provider.ClaudeTokenSource, grokToken *provider.GrokTokenSource, ag *provideradapter.AntigravityProbe, transport httpclient.UpstreamTransport, cfg *config.Config, profiles *egressprovider.TLSProfiles, routers *egress.TLSFingerprintRouterService, settings *gateway.RuntimeSettings, tasks *provideradapter.ProbeTasks, manager *lifecycle.Manager, openaiTest *provideradapter.OpenAIProviderTest) *provider.TestService {
	urlPolicy := egress.OperatorURLPolicy{Enabled: cfg.Security.URLAllowlist.Enabled, AllowInsecureHTTP: cfg.Security.URLAllowlist.AllowInsecureHTTP, AllowPrivateHosts: cfg.Security.URLAllowlist.AllowPrivateHosts, UpstreamHosts: slices.Clone(cfg.Security.URLAllowlist.UpstreamHosts)}
	geminiTest := &provideradapter.GeminiProviderTest{Tokens: geminiToken, Transport: transport, Profiles: profiles, ValidateURL: urlPolicy.Validate}
	anthropicTest := &provideradapter.AnthropicProviderTest{Tokens: claudeToken, Transport: transport, Profiles: profiles, Store: store, ValidateURL: urlPolicy.Validate}
	// 保留管理测试独立的会话作用域，并明确登记其停止拥有者。
	qoderSessions := provideradapter.NewQoderTokenProvider(qoder.SessionBuilder{})
	qoderSessions.SetHTTPUpstream(transport, profiles)
	manager.Register(lifecycle.Hook{Name: "ProviderTestQoderSessions", StopOrder: 26, Stop: qoderSessions.StopContext})
	targets := &provideradapter.TestTargets{
		Read: store.GetByID, OpenAI: openaiTest, Gemini: geminiTest, Anthropic: anthropicTest,
		Qoder: &provideradapter.QoderProviderTest{Sessions: qoderSessions, Client: qoder.NewClient(qoder.APIBaseURL), Transport: transport, Profiles: profiles, RewriteModel: openaiprotocol.ReplaceModelInBody},
		Grok:  &provideradapter.GrokProviderTest{Tokens: grokToken, Transport: transport, Store: store, OperatorValidator: urlPolicy.Validate, DefaultBaseURL: gatewayprovider.GrokDefaultBaseURLReader(settings)},
		CN:    &provideradapter.CNProviderTest{Transport: transport, Profiles: profiles, Store: store, ValidateURL: urlPolicy.Validate, Responses: openaiTest},
		Antigravity: &provideradapter.AntigravityProviderTest{Gemini: geminiTest, Anthropic: anthropicTest, Probe: func(ctx context.Context, value *provider.Record, request provider.PreparedTestRequest) (*antigravity.TestConnectionResult, error) {
			return ag.Execute(ctx, value, request)
		}},
	}
	core := provider.NewTestService(targets, provider.TestOptions{Now: time.Now, Error: func(message string) { log.Printf("Provider test error: %s", message) }, WriteError: func(err error) { log.Printf("failed to write SSE event: %v", err) }})
	return core
}

// 普通测试与题目检测复用相同凭据、TLS 和请求构造器；严格回答校验由检测请求显式注入。
func provideOpenAITestExecutor(store *providerpostgres.ProviderStore, transport httpclient.UpstreamTransport, cfg *config.Config, profiles *egressprovider.TLSProfiles, routers *egress.TLSFingerprintRouterService, settings *gateway.RuntimeSettings, tasks *provideradapter.ProbeTasks) *provideradapter.OpenAIProviderTest {
	urlPolicy := egress.OperatorURLPolicy{Enabled: cfg.Security.URLAllowlist.Enabled, AllowInsecureHTTP: cfg.Security.URLAllowlist.AllowInsecureHTTP, AllowPrivateHosts: cfg.Security.URLAllowlist.AllowPrivateHosts, UpstreamHosts: slices.Clone(cfg.Security.URLAllowlist.UpstreamHosts)}
	policy := &provideradapter.OpenAIProbePolicy{Available: true, ForceCLI: cfg.Gateway.ForceCodexCLI, Read: store.GetByID, AllowClaudeCode: settings.IsOpenAIAllowClaudeCodeCodexPluginEnabled, BrowserUserAgent: settings.GetOpenAICodexUserAgent, DefaultBrowserUserAgent: gateway.DefaultOpenAICodexUserAgent, Routers: routers, Profiles: profiles, ManualProfiles: profiles}
	return &provideradapter.OpenAIProviderTest{Store: store, Transport: transport, ValidateURL: urlPolicy.Validate, Prepare: policy.Prepare, ApplyRouting: policy.ApplyTestRouting, ResolveTLS: policy.ResolveTestTLS, EnsureTask: tasks.Ensure}
}

// provideProviderTestHTTP 与后台复用唯一测试用例，成功恢复仍调用原健康端口。
func provideProviderTestHTTP(core *provider.TestService, recovery *provider.RecoveryService) *providerhttp.TestHandler {
	return providerhttp.NewTestHandler(core, func(ctx context.Context, id int64) error {
		_, err := recovery.RecoverProviderAfterSuccessfulTest(ctx, id)
		return err
	})
}
