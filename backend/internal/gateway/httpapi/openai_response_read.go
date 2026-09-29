package httpapi

import (
	"context"
	"net/http"
	"time"

	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/gin-gonic/gin"
)

// ReadStreamObservation 保留结果与错误并存；完成资格仍由入口用例决定。
func (p *OpenAIResponseOutput) ReadStreamObservation(ctx context.Context, resp *http.Response, c *gin.Context, target *gatewayadapter.ExecutionProvider, started time.Time, original, mapped, effort string) (*openai.StreamingResult, error) {
	return openai.ReadStreamingResponse(ctx, resp, upstream.NewOutputContext(ResponseSink{Writer: c.Writer}), p.StreamOptions(ctx, c, target, effort), started, original, mapped, effort)
}

func (p *OpenAIResponseOutput) NonStream(ctx context.Context, resp *http.Response, c *gin.Context, target *gatewayadapter.ExecutionProvider, original, mapped string) (*openai.NonStreamingResult, error) {
	return openai.ReadNonStreamingResponse(ctx, resp, upstream.NewOutputContext(ResponseSink{Writer: c.Writer}), p.NonStreamOptions(ctx, c, target), original, mapped)
}
