package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider/postgres"
)

// provideOpenAITokens 绑定同一刷新协调器、缓存和指标，运行阻断端口在网关构造时接入。
func provideOpenAITokens(store *postgres.ProviderStore, cache provider.AccessTokenCache, authorization *provider.OpenAIAuthorization, refresh *provider.OAuthRefreshAPI, blocks *provider.RuntimeBlockState) *provider.OpenAITokenSource {
	executor := &provider.OpenAITokenRefresher{Authorization: authorization}
	return &provider.OpenAITokenSource{
		Cache: cache,
		Block: func(record *provider.Record, until time.Time, reason string) {
			if record != nil && (record.Platform == capability.PlatformOpenAI || record.Platform == capability.PlatformGrok) {
				blocks.Block(record.ID, until, reason)
			}
		},
		Repository: store,
		SetError:   store.SetError,
		Metrics:    &provider.OpenAITokenMetricsStore{},
		Policy:     provider.OpenAIProviderRefreshPolicy(),
		Debug:      slog.Debug,
		Warn:       slog.Warn,
		Refresh: func(ctx context.Context, record *provider.Record, window time.Duration) (*provider.OAuthRefreshResult, error) {
			return refresh.RefreshIfNeeded(ctx, record, executor, window)
		},
	}
}

// provideOpenAIExecutionCredentials 复用原持久读取和两种 token 源，不提前解析影子或读取凭据。
func provideOpenAIExecutionCredentials(store *postgres.ProviderStore, openai *provider.OpenAITokenSource, grok *provider.GrokTokenSource) *provider.OpenAIExecutionCredentials {
	out := &provider.OpenAIExecutionCredentials{}
	if store != nil {
		out.Parent = store.GetByID
	}
	if openai != nil {
		out.OpenAI = openai.GetAccessToken
	}
	if grok != nil {
		out.Grok = grok.GetAccessToken
	}
	return out
}
