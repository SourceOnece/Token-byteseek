package app

import (
	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/gateway"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
)

// provideOpenAIAuthorization 直接构造唯一授权实例，动态 UA 仍通过原设置读取器获取。
func provideOpenAIAuthorization(proxies egress.ProxyRepository, client provideradapter.OpenAIOAuthClient, privacy openai.PrivacyClientFactory, settings *gateway.RuntimeSettings, routers provideradapter.OpenAITokenRouterReader, profiles provideradapter.OpenAITokenProfileResolver) *provider.OpenAIAuthorization {
	deps := &provideradapter.OpenAIAuthorizationDependencies{
		Proxies:        proxies,
		Client:         client,
		PrivacyFactory: privacy,
		Routers:        routers,
		Profiles:       profiles,
	}
	if settings != nil {
		deps.CodexUserAgent = settings.GetOpenAICodexUserAgent
	}
	return provider.NewOpenAIAuthorization(provider.NewOpenAISessionStore(), provideradapter.OpenAIAuthorizationOptions(deps))
}
