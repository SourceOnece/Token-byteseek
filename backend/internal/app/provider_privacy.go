package app

import (
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"

	egresspostgres "github.com/TokenFlux/TokenRouter/internal/egress/postgres"
)

// provideProviderPrivacy 统一管理与后台刷新使用的隐私用例，平台交换保留原技术客户端。
func provideProviderPrivacy(store *providerpostgres.ProviderStore, proxies *egresspostgres.ProxyStore, factory openai.PrivacyClientFactory, manager *lifecycle.Manager) *provider.PrivacyService {
	core := provider.NewPrivacyService(store, proxies, provideradapter.PrivacyOptions(factory, openai.PrivacyEndpoints{}))
	manager.Register(lifecycle.Hook{Name: "ProviderPrivacy", StopOrder: 26, Stop: core.StopContext})
	return core
}
