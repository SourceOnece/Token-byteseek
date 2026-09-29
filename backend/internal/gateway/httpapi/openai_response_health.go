package httpapi

import (
	"context"
	"net/http"

	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// ApplyHTTPFailure 保留旧 HTTP 失败入口只使用首个显式模型的边界。
func (p *OpenAIResponseOutput) ApplyHTTPFailure(ctx context.Context, resp *http.Response, target *gatewayadapter.ExecutionProvider, body []byte, models ...string) provider.UpstreamErrorDecision {
	if len(models) > 0 {
		return gatewayadapter.ApplyOpenAIResponseHealth(ctx, p.Health, target, resp.StatusCode, resp.Header, body, false, models[0])
	}
	return gatewayadapter.ApplyOpenAIResponseHealth(ctx, p.Health, target, resp.StatusCode, resp.Header, body, false)
}
