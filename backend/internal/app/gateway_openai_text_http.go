package app

import (
	"context"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/gateway/httpapi/openaiattempt"
	textflow "github.com/TokenFlux/TokenRouter/internal/gateway/text"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/gateway/admission"
	"github.com/TokenFlux/TokenRouter/internal/gateway/errorpolicy"
	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	"github.com/TokenFlux/TokenRouter/internal/gateway/promptpolicy"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
	"github.com/TokenFlux/TokenRouter/internal/moderation"
	openaiwire "github.com/TokenFlux/TokenRouter/internal/protocol/openai"
	"github.com/TokenFlux/TokenRouter/internal/routing"
)

// provideOpenAITextHTTP 直接组合原生 HTTP 与固定单次运行时，不经过旧 Handler。
func provideOpenAITextHTTP(
	source *gatewayhttp.OpenAIResponsesExecutor,
	funding *admission.FundingAdmission,
	keys *apikey.APIKeyService,
	resources *gatewayhttp.OpenAIHTTPResources,
	cyber *gatewayhttp.CyberHandler,
	rules *errorpolicy.ErrorPassthroughService,
	moderator *moderation.ContentModerationService,
	prompts *promptpolicy.Service,
	cfg *config.Config,
	runtime *openaiattempt.Runtime,
	activity *gatewayRequestActivity,
	planner *gatewayadapter.RoutePlanner, cache session.GatewayCache,
	messages *messageHTTPBindings,
	subscriptions *billing.SubscriptionService,
) *gatewayhttp.OpenAITextHandler {
	options := openAITextOptions(cfg)
	bindings := openAITextBindings(source, funding, keys, resources, cyber, rules, moderator, planner, cache, subscriptions)
	if messages != nil {
		bindings.ClientVersions = messages.bindings.ClientVersions
	}
	result := gatewayhttp.NewBoundOpenAITextHandler(options, bindings, prompts, textflow.NewResponsesExecutor(runtime, textflow.ResponseOptions{MaxSwitches: options.MaxSwitches}, textflow.ResponseOptions{MaxSwitches: options.MaxSwitches, FirstOutputBudget: true}))
	result.BindRequestActivity(activity.Enter)
	return result
}

// openAITextOptions 仅投影静态 HTTP 与切号预算。
func openAITextOptions(cfg *config.Config) gatewayhttp.OpenAITextOptions {
	options := gatewayhttp.OpenAITextOptions{MaxSwitches: 3}
	if cfg != nil {
		options.ForceCodexCLI = cfg.Gateway.ForceCodexCLI
		options.MaxBodyBytes = cfg.Gateway.MaxBodySize
		if cfg.Gateway.MaxProviderSwitches > 0 {
			options.MaxSwitches = cfg.Gateway.MaxProviderSwitches
		}
		if cfg.Gateway.StreamKeepaliveInterval > 0 {
			options.CompactKeepaliveInterval = time.Duration(cfg.Gateway.StreamKeepaliveInterval) * time.Second
		}
	}
	return options
}

// openAITextBindings 固定原生能力，运行时只创建请求数据。
func openAITextBindings(source *gatewayhttp.OpenAIResponsesExecutor, funding *admission.FundingAdmission, keys *apikey.APIKeyService, resources *gatewayhttp.OpenAIHTTPResources, cyber *gatewayhttp.CyberHandler, rules *errorpolicy.ErrorPassthroughService, moderator *moderation.ContentModerationService, planner *gatewayadapter.RoutePlanner, cache session.GatewayCache, subscriptions *billing.SubscriptionService) gatewayhttp.OpenAITextBindings {
	var moderationPort gatewayhttp.ModerationPort
	if moderator != nil {
		moderationPort = moderator
	}
	bindings := gatewayhttp.OpenAITextBindings{
		Dependencies: gatewayhttp.OpenAIDependencies{
			Handler:     true,
			Gateway:     source != nil,
			Funding:     funding != nil,
			Keys:        keys != nil,
			Concurrency: resources != nil && resources.Concurrency != nil && resources.Concurrency.Service() != nil,
		},
		Resources:     resources,
		ResponseOwner: func() session.HTTPResponseOwnerReader { return source.Lineage.Store },
		Moderation:    moderationPort,
		PlanRoute: func(ctx context.Context, key *apikey.APIKey, model string) routing.RoutePlan {
			return planner.PlanKey(ctx, key, model)
		},
		ReplaceModel:        openaiwire.ReplaceModelInBody,
		ClientGroupFallback: provideClientGroupFallbackResolver(keys, funding, subscriptions, cache),
		Errors:              rules,
		Funding:             funding,
		Cyber:               cyber,
		IsolateSession:      messageSessionIsolation(cache),
	}
	return bindings
}
