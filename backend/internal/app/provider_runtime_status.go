package app

import (
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	usagepostgres "github.com/TokenFlux/TokenRouter/internal/usage/postgres"

	"github.com/TokenFlux/TokenRouter/internal/provider"

	providerhttp "github.com/TokenFlux/TokenRouter/internal/provider/httpapi"

	"github.com/TokenFlux/TokenRouter/internal/config"
	settingscore "github.com/TokenFlux/TokenRouter/internal/settings"
)

// provideProviderRuntimePresenter 直接绑定新展示实现，不再引用旧 handler。
func provideProviderRuntimePresenter(admin *provider.Admin, ollama *provider.OllamaCloudUsageService, concurrency *scheduler.ConcurrencyService, usage *usagepostgres.Store, sessions scheduler.SessionLimitCache, rpm scheduler.RPMCache, settings *provider.QuotaSettingsCache) *providerhttp.RuntimePresenter {
	return providerhttp.NewRuntimePresenter(provider.NewRuntimeStatusReader(provideradapter.RuntimeStatusOptions(concurrency, usage, sessions, rpm, settings)), admin, ollama)
}

// provideProviderManagementList 复用当前调度反馈和技术读取实例，保持列表批量查询。
func provideProviderManagementList(admin *provider.Admin, ollama *provider.OllamaCloudUsageService, concurrency *scheduler.ConcurrencyService, usage *usagepostgres.Store, sessions scheduler.SessionLimitCache, rpm scheduler.RPMCache, settings *provider.QuotaSettingsCache, shared *schedulerSharedState, store *settingscore.Store, cfg *config.Config) *provider.ManagementList {
	return provider.NewManagementList(admin, provider.NewRuntimeStatusReader(provideradapter.RuntimeStatusOptions(concurrency, usage, sessions, rpm, settings)), provider.NewSchedulerScoreView(admin, providerScoreOptions(concurrency, shared, store, cfg)), ollama)
}
