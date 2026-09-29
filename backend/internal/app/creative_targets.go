package app

import (
	"github.com/TokenFlux/TokenRouter/internal/config"
	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// provideCreativeTargets 与 HTTP 入口共享凭据、身份和传输；任务不构造另一份网关状态。
func provideCreativeTargets(cfg *config.Config, auxiliary *gatewayhttp.OpenAIAuxiliary, tokens *provider.GeminiTokenSource, activity *gatewayRequestActivity) *gatewayadapter.CreativeTargets {
	requests := auxiliary.Requests
	return &gatewayadapter.CreativeTargets{
		Requests:     requests,
		Credentials:  requests.Credentials,
		Identity:     requests.Identity,
		Transport:    requests.Transport,
		Routes:       provideGrokRoutes(cfg, nil),
		GeminiTokens: tokens,
		Enter:        activity.Enter,
	}
}
