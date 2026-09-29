//go:build unit

package provider_test

import (
	"context"
	"log/slog"
	"time"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// 旧网关测试暂用仓储端口投影，令牌规则仍只执行原生提供商实现。
func newGrokTokenSourceForTest(repo gatewayprovider.ExecutionProviderStore, cache provider.AccessTokenCache) *provider.GrokTokenSource {
	return &provider.GrokTokenSource{Repository: tokenSourceFixtureRepository(repo), Cache: cache, Policy: provider.GrokProviderRefreshPolicy()}
}

// 仅连接已有测试刷新器，不复制锁、凭据合并或错误处理。
func bindGrokRefreshForTest(source *provider.GrokTokenSource, refresh *provider.OAuthRefreshAPI, executor provider.OAuthRefreshExecutor) {
	source.Refresh = func(ctx context.Context, record *provider.Record, window time.Duration) (*provider.OAuthRefreshResult, error) {
		return refresh.RefreshIfNeeded(ctx, record, executor, window)
	}
}

// newGrokCredentialRefreshForTest 只配置原生协调器，不保留旧结果或执行器包装。
func newGrokCredentialRefreshForTest(repo gatewayprovider.ExecutionProviderStore, cache provider.AccessTokenCache) *provider.OAuthRefreshAPI {
	return provider.NewOAuthRefreshAPI(tokenSourceFixtureRepository(repo), cache, provider.RefreshOptions{Now: time.Now, Warn: slog.Warn, Info: slog.Info, Error: slog.Error, Platform: provider.ProviderRefreshPlatformPolicy()})
}
