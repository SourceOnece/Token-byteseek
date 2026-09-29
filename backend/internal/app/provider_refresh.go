package app

import (
	"log/slog"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
)

// provideProviderRefresh 为所有旧平台消费者绑定同一协调器与提供商条件写入实现。
func provideProviderRefresh(store *providerpostgres.ProviderStore, cache provider.AccessTokenCache, manager *lifecycle.Manager) *provider.OAuthRefreshAPI {
	coordinator := provider.NewOAuthRefreshAPI(store, cache, provider.RefreshOptions{
		Now: time.Now, Warn: slog.Warn, Info: slog.Info, Error: slog.Error,
		Platform: provider.ProviderRefreshPlatformPolicy(),
	})
	manager.Register(lifecycle.Hook{Name: "ProviderRefreshCoordinator", StopOrder: 25, Stop: coordinator.StopContext})
	return coordinator
}
