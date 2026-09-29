package app

import (
	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// provideProviderTier 使用同一个提供商配置用例，停止纳入原后台预算。
func provideProviderTier(admin *provider.Admin, source *provider.GeminiAuthorization, manager *lifecycle.Manager) *provider.TierManagement {
	core := provider.NewTierManagement(admin, provider.ProviderTierManagementOptions(source))
	manager.Register(lifecycle.Hook{Name: "ProviderTier", StopOrder: 26, Stop: core.StopContext})
	return core
}
