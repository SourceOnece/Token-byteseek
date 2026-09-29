package provider

import (
	"context"
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// ApplyGrokExecutionHealth 只投影当前尝试和已解码提供商，原有写入仍由提供商模块执行。
func ApplyGrokExecutionHealth(ctx context.Context, health *provideradapter.GrokHealth, target *ExecutionProvider, status int, headers http.Header, body []byte, teamModel string, models ...string) provider.UpstreamErrorDecision {
	if health == nil || target == nil {
		return provider.UpstreamErrorDecision{Policy: provider.ErrorPolicyNone}
	}
	return health.ObserveError(ctx, target.View(), provideradapter.GrokHealthInput{
		Observation:     HealthObservationFromContext(ctx, status, headers, body, models),
		Models:          models,
		QuotaModel:      teamModel,
		TeamModel:       teamModel,
		RequestScoped:   IsRequestScopedProviderFailure(target, status, body),
		ServerTransient: IsTransientProviderFailure(status, body),
	})
}
