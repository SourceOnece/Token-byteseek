package app

import (
	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// provideQoderRequestRefresh 为所有 Qoder 入站链绑定同一协调器、原生存储和会话缓存。
func provideQoderRequestRefresh(store *providerpostgres.ProviderStore, tokens *provideradapter.QoderTokenProvider, coordinator *provider.OAuthRefreshAPI, transport provideradapter.QoderTransport, profiles *egressprovider.TLSProfiles) *provideradapter.QoderRequestRefresh {
	return &provideradapter.QoderRequestRefresh{Store: store, Tokens: tokens, Coordinator: coordinator, Transport: transport, Profiles: profiles}
}

// provideQoderRuntime 绑定同一令牌源、传输池与提供商存储，会话及执行器由原生运行时独占。
func provideQoderRuntime(tokens *provideradapter.QoderTokenProvider, transport provideradapter.QoderTransport, profiles *egressprovider.TLSProfiles, store *providerpostgres.ProviderStore) *gatewayprovider.QoderRuntime {
	return gatewayprovider.NewQoderRuntime(gatewayprovider.QoderRuntimeOptions{Tokens: tokens, Transport: transport, Profiles: profiles, Health: store})
}
