package app

import (
	"context"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	"github.com/TokenFlux/TokenRouter/internal/codexticket"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	"github.com/TokenFlux/TokenRouter/internal/gateway/promptpolicy"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/selection"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// provideExecutionAgentIdentity 与所有提供商查询共用协调器和连接失效拥有者。
func provideExecutionAgentIdentity(tasks *provider.OpenAITaskCoordinator, store gatewayadapter.ExecutionProviderStore, connections *gatewayhttp.OpenAIWSConnections) *gatewayadapter.ExecutionAgentIdentity {
	return gatewayadapter.NewExecutionAgentIdentity(tasks, store, func(ctx context.Context, value *provider.Record) (string, error) {
		return provideradapter.RegisterAgentIdentityTask(ctx, value, "https://auth.openai.com/api/accounts")
	}, connections.InvalidateProvider)
}

func provideAnthropicPromptCache() *session.AnthropicPromptCache {
	return session.NewAnthropicPromptCache(time.Now)
}

// provideOpenAITextExecutor 构造独立的文本能力，共用请求、输出和后台观察实例。
func provideOpenAITextExecutor(cfg *config.Config, store gatewayadapter.ExecutionProviderStore, identity *gatewayadapter.ExecutionAgentIdentity, credentials *provider.OpenAIExecutionCredentials, transport httpclient.UpstreamTransport, profiles *egressprovider.TLSProfiles, routers *egress.TLSFingerprintRouterService, readers *gatewayadapter.RuntimeReaders, grok *gatewayhttp.GrokExecutor, output *gatewayhttp.OpenAIResponseOutput, cache *session.AnthropicPromptCache, choices *selection.Compatible, compact *gatewayhttp.CompactExecutor, tasks *lifecycle.Tasks, tickets *codexticket.CodexTicketService) *gatewayhttp.OpenAITextExecutor {
	text := openAITextExecution(cfg, store, identity, credentials, transport, profiles, routers, readers, grok, output, cache, choices.OpenAIHTTPResponseStickyTTL, compact)
	text.CodexUsage.Go = tasks.Go
	text.Requests.Tickets = tickets
	return text
}

func provideOpenAIImageBridgePolicy(cfg *config.Config) *gatewayadapter.ResponseImagePolicy {
	policy := &gatewayadapter.ResponseImagePolicy{}
	if cfg != nil {
		policy.DefaultEnabled = cfg.Gateway.CodexImageGenerationBridgeEnabled
	}
	return policy
}

func provideOpenAIEncryptedLineage(state session.OpenAIWSStateStore, choices *selection.Compatible) *gatewayhttp.OpenAIEncryptedLineage {
	return &gatewayhttp.OpenAIEncryptedLineage{Store: state, TTL: choices.SessionStickyTTL}
}

func provideOpenAIWebSockets(cfg *config.Config, connections *gatewayhttp.OpenAIWSConnections, text *gatewayhttp.OpenAITextExecutor, prompts *promptpolicy.Service, choices *selection.Compatible, lineage *gatewayhttp.OpenAIEncryptedLineage, imagePolicy *gatewayadapter.ResponseImagePolicy, cache session.GatewayCache) *gatewayhttp.OpenAIWebSocketExecutor {
	return gatewayhttp.NewOpenAIWebSocketExecutor(gatewayhttp.OpenAIWSDependencies{Options: openAIWSExecutionOptions(cfg), Connections: connections, Requests: text.Requests, Output: text.Output, Grok: text.Grok, FastPolicy: text.FastPolicy, Prompts: prompts, Selection: choices, State: lineage.Store, Lineage: lineage, ImageBridge: imagePolicy, Cache: cache})
}

func provideOpenAIResponses(text *gatewayhttp.OpenAITextExecutor, sockets *gatewayhttp.OpenAIWebSocketExecutor, choices *selection.Compatible, lineage *gatewayhttp.OpenAIEncryptedLineage, imagePolicy *gatewayadapter.ResponseImagePolicy) *gatewayhttp.OpenAIResponsesExecutor {
	return &gatewayhttp.OpenAIResponsesExecutor{Requests: text.Requests, Output: text.Output, Text: text, Grok: text.Grok, Lineage: lineage, ImageBridge: imagePolicy, ResolveTransport: choices.ResolveTransport, WebSocket: sockets.ForwardHTTPWebSocket}
}
