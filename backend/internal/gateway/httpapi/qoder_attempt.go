package httpapi

import (
	"context"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/gateway/forward"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/protocol"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
	"github.com/gin-gonic/gin"
)

// ForwardQoderAttempt 只执行单次尝试并同步写出，提供商切换仍由外层唯一循环决定。
// 保留旧入口的部分结果资格和结果字段，不额外填充首次输出或估算用量。
func ForwardQoderAttempt(ctx context.Context, c *gin.Context, runtime *gatewayadapter.QoderRuntime, value *provider.Record, body []byte, wire protocol.ProtocolID, responseModels ...string) (*forward.MessagesResult, error) {
	responseModel := qoder.FirstNonEmptyQoder(responseModels...)
	if responseModel == "" {
		responseModel = strings.TrimSpace(qoder.GjsonString(body, "model"))
	}
	executor, input := runtime.PrepareQoderTarget(QoderRequestMetadata(c), value, body, wire, responseModel)
	result, err := executor.Execute(ctx, input, ResponseSink{Writer: c.Writer})
	if err != nil {
		runtime.ObserveQoderFailure(ctx, value, err)
		if !result.Served || !result.HasUsage {
			return nil, err
		}
	}
	return &forward.MessagesResult{RequestID: result.RequestID, Model: result.Model, UpstreamModel: result.UpstreamModel, Usage: result.Usage, Stream: result.Stream, Duration: result.Duration, ClientDisconnect: result.ClientDisconnect}, err
}
