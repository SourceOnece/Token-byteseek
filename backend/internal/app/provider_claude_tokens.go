package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// provideClaudeTokens 保留 OAuth 与 Vertex 两条获取路径的原缓存和调用时点。
func provideClaudeTokens(store *postgres.ProviderStore, cache provider.AccessTokenCache, authorization *provider.ClaudeAuthorization, refresh *provider.OAuthRefreshAPI) *provider.ClaudeTokenSource {
	executor := &provider.ClaudeTokenRefresher{Authorization: authorization}
	return &provider.ClaudeTokenSource{Options: provider.ClaudeTokenOptions{
		Cache: cache, Repository: store, Policy: provider.ClaudeProviderRefreshPolicy(),
		Debug: slog.Debug, Warn: slog.Warn,
		Vertex: func(ctx context.Context, value *provider.Record) (string, error) {
			return provideradapter.VertexServiceAccountAccessToken(ctx, cache, value)
		},
		Refresh: func(ctx context.Context, value *provider.Record, window time.Duration) (*provider.OAuthRefreshResult, error) {
			return refresh.RefreshIfNeeded(ctx, value, executor, window)
		},
	}}
}

// provideMessageCredentials 固定复用原 Claude/Vertex 源，其他平台保持存量凭据读取。
func provideMessageCredentials(claude *provider.ClaudeTokenSource) *provider.MessageCredentialSource {
	result := &provider.MessageCredentialSource{}
	if claude != nil {
		result.Claude = claude.GetAccessToken
	}
	return result
}
