package app

import (
	"context"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider/postgres"
)

// provideGrokTokens 让请求与手动查询共用原生缓存及刷新协调器。
func provideGrokTokens(store *postgres.ProviderStore, cache provider.AccessTokenCache, authorization *provider.GrokAuthorization, refresh *provider.OAuthRefreshAPI) *provider.GrokTokenSource {
	executor := provider.NewGrokTokenRefresher(authorization)
	return &provider.GrokTokenSource{
		Cache:      cache,
		Repository: store,
		Policy:     provider.GrokProviderRefreshPolicy(),
		Refresh: func(ctx context.Context, record *provider.Record, window time.Duration) (*provider.OAuthRefreshResult, error) {
			return refresh.RefreshIfNeeded(ctx, record, executor, window)
		},
	}
}
