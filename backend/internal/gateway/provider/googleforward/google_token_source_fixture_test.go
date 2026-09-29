package googleforward_test

import (
	"context"
	"log/slog"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// 旧执行链夹具直接使用原生源，技术依赖按原零配置构造注入。
func newGeminiTokenSourceForTest() *provider.GeminiTokenSource {
	return &provider.GeminiTokenSource{Options: provider.GeminiTokenOptions{
		Debug: slog.Debug,
		Warn:  slog.Warn,

		Vertex: func(ctx context.Context, value *provider.Record) (string, error) {
			return provideradapter.VertexServiceAccountAccessToken(ctx, nil, value)
		},
	}}
}

func newAntigravityTokenSourceForTest(cache provider.AccessTokenCache) *provider.AntigravityTokenSource {
	return &provider.AntigravityTokenSource{Options: provider.AntigravityTokenOptions{
		Cache: cache, Debug: slog.Debug, Warn: slog.Warn,
	}}
}
