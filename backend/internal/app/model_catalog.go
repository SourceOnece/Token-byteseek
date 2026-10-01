package app

import (
	"slices"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/modelidentity"
	"github.com/TokenFlux/TokenRouter/internal/modelcatalog/provider"
)

// provideModelCatalogService 从同一份 bootstrap 配置投影技术参数，构造期间不启动任务。
// 初始化、周期更新和停止继续由既有 ModelCatalogInitialization/ModelCatalogService hook 唯一管理。
func provideModelCatalogService(cfg *config.Config, remote provider.RemoteClient) (*provider.Service, error) {
	options := provider.Options{
		DataDir:               cfg.Pricing.DataDir,
		RemoteURL:             cfg.Pricing.RemoteURL,
		FallbackFile:          cfg.Pricing.FallbackFile,
		OverrideFile:          cfg.Pricing.OverrideFile,
		CheckIntervalMinutes:  cfg.Pricing.CheckIntervalMinutes,
		URLAllowlistEnabled:   cfg.Security.URLAllowlist.Enabled,
		AllowInsecureHTTP:     cfg.Security.URLAllowlist.AllowInsecureHTTP,
		AllowPrivateHosts:     cfg.Security.URLAllowlist.AllowPrivateHosts,
		PricingHosts:          slices.Clone(cfg.Security.URLAllowlist.PricingHosts),
		ModelLookupCandidates: modelidentity.CandidatesFactory,
	}
	return provider.NewService(options, remote), nil
}

// provideModelCatalogRemoteClient 沿用更新代理及显式直连回退配置。
func provideModelCatalogRemoteClient(cfg *config.Config) provider.RemoteClient {
	return provider.NewRemoteClient(cfg.Update.ProxyURL, cfg.Security.ProxyFallback.AllowDirectOnError)
}
