package app

import (
	"slices"

	"github.com/TokenFlux/TokenRouter/internal/gateway"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/gateway/forward"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	usagepostgres "github.com/TokenFlux/TokenRouter/internal/usage/postgres"
)

// provideGrokQuota 直接绑定唯一探测运行时、原生提供商存储及供应商端口。
func provideGrokQuota(store *providerpostgres.ProviderStore, proxies egress.ProxyRepository, token *provider.GrokTokenSource, transport httpclient.UpstreamTransport, cfg *config.Config, usageStore *usagepostgres.Store, settings *gateway.RuntimeSettings) *provider.GrokQuotaService {
	policy := egress.OperatorURLPolicy{Enabled: cfg.Security.URLAllowlist.Enabled, AllowInsecureHTTP: cfg.Security.URLAllowlist.AllowInsecureHTTP, AllowPrivateHosts: cfg.Security.URLAllowlist.AllowPrivateHosts, UpstreamHosts: slices.Clone(cfg.Security.URLAllowlist.UpstreamHosts)}
	requests := &provideradapter.GrokQuotaTransport{Do: transport.Do, Proxy: proxies.GetByID, DefaultBaseURL: gatewayprovider.GrokDefaultBaseURLReader(settings), OperatorValidator: policy.Validate, MapStatus: forward.MapStatus}

	return provideradapter.NewGrokQuota(store, token, requests, newProviderLocalUsageStats(usageStore))
}
