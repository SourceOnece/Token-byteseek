package app

import (
	"time"

	egressadapter "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/provider"

	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	providerhttp "github.com/TokenFlux/TokenRouter/internal/provider/httpapi"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
)

// provideUpstreamUsage 将唯一提供商查询核心接到同一存储、平台执行和有界退出。
func provideUpstreamUsage(store *providerpostgres.ProviderStore, upstream httpclient.UpstreamTransport, cfg *config.Config, tls *egressadapter.TLSProfiles, manager *lifecycle.Manager) *provider.UpstreamUsageService {
	options := provideradapter.UsageHTTPOptions{Available: store != nil && upstream != nil}
	if cfg != nil {
		value := cfg.Security.URLAllowlist
		options.Policy = egress.UsageURLPolicy{Configured: true, Enabled: value.Enabled, AllowInsecureHTTP: value.AllowInsecureHTTP, AllowPrivateHosts: value.AllowPrivateHosts, UpstreamHosts: value.UpstreamHosts}
	}
	if upstream != nil {
		options.Do = upstream.DoWithTLS
	}
	if tls != nil {
		options.ResolveTLS = tls.ResolveRequestTLS
	}
	source := provideradapter.NewUpstreamUsageHTTPExecution(options)
	core := provider.NewUpstreamUsageService(store, source, provider.UpstreamUsageOptions{Now: time.Now})
	// 周期生产者停止后，查询完成才允许后续 SQL/HTTP 依赖释放。
	manager.Register(lifecycle.Hook{Name: "ProviderUpstreamUsage", StopOrder: 25, Stop: core.StopContext})
	return core
}

// provideUpstreamUsageHTTP 将提供商用量查询用例绑定到 HTTP 接口。
func provideUpstreamUsageHTTP(source *provider.UpstreamUsageService) *providerhttp.UpstreamUsageHandler {
	return providerhttp.NewUpstreamUsageHandler(source)
}
