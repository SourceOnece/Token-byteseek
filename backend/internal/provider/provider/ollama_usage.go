package provider

import (
	"context"
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/TokenFlux/TokenRouter/internal/upstream/ollama"
)

// OllamaUsageFetcher 只投影受控 Cookie 和请求参数，复用原生客户端及唯一 HTTP 池。
func OllamaUsageFetcher(do func(*http.Request, string, int64, int) (*http.Response, error)) func(context.Context, provider.OllamaUsageFetchInput) (*provider.OllamaUsageObservation, error) {
	return func(ctx context.Context, input provider.OllamaUsageFetchInput) (*provider.OllamaUsageObservation, error) {
		if do == nil {
			return nil, provider.ErrOllamaCloudUsageUnavailable
		}
		return ollama.FetchUsage(ctx, ollama.FetchInput{ObservedAt: input.ObservedAt, Cookie: input.Cookie}, ollama.FetchOptions{
			Do: func(req *http.Request) (*http.Response, error) {
				return do(req, input.ProxyURL, input.ProviderID, input.Concurrency)
			},
			Context:     upstream.WithHTTPUpstreamRedirectsDisabled,
			Unavailable: provider.ErrOllamaCloudUsageUnavailable,
		})
	}
}
