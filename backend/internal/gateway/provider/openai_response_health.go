package provider

import (
	"context"
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
)

// ApplyOpenAIResponseHealth 固化请求范围与模型观测，保留默认状态处理的字段写回边界。
func ApplyOpenAIResponseHealth(ctx context.Context, health *provideradapter.OpenAIResponseHealth, target *ExecutionProvider, status int, headers http.Header, body []byte, suppressDefaultRateLimit bool, models ...string) provider.UpstreamErrorDecision {
	return health.Apply(ctx, target.View(), provideradapter.OpenAIResponseHealthInput{
		Observation:              HealthObservationFromContext(ctx, status, headers, body, models),
		Models:                   models,
		SuppressDefaultRateLimit: suppressDefaultRateLimit,
		ContentRejected:          IsContentPolicyRejection(body) || IsOpenAICyberWarningPayload(body, upstream.ExtractErrorMessage(body)),
		RequestScoped:            IsRequestScopedProviderFailure(target, status, body),
		SelfBuiltImage:           openai.IsOpenAIImagesSelfBuiltRequest(ctx),
		Transient:                IsTransientProviderFailure(status, body),
	})
}
