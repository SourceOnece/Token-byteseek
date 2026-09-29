package provider

import (
	"context"
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/gateway/requeststate"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// HealthObservationFromContext 在网关边界固化模型、thinking 和端点意图，提供商观测不回读业务 context。
func HealthObservationFromContext(ctx context.Context, status int, headers http.Header, body []byte, models []string) provideradapter.HealthObservation {
	input := provideradapter.HealthObservation{Status: status, Headers: headers, Body: body, EffectiveModel: requeststate.HealthModel(ctx, models), ModelProvided: len(models) > 0, Thinking: requeststate.HealthThinking(ctx), ImagesEndpoint: requeststate.OpenAIImagesEndpointFromContext(ctx)}
	if len(models) > 0 {
		input.Model = models[0]
	}
	return input
}

// ApplyExecutionHealth 保留观测使用独立记录、返回后仅回写凭据与附加状态的边界。
func ApplyExecutionHealth(ctx context.Context, observer *provideradapter.UpstreamHealth, target *ExecutionProvider, input provideradapter.HealthObservation) provider.UpstreamErrorDecision {
	record := ExecutionRecord(target)
	result := observer.ApplyUpstreamError(ctx, record, input)
	if target != nil && record != nil {
		target.Record.Credentials, target.Record.Extra = record.Credentials, record.Extra
	}
	return result
}

// ExecutionErrorPolicy 只借用同步裁决所需的策略字段，不复制或持有完整提供商。
func ExecutionErrorPolicy(value *ExecutionProvider) *provider.Record {
	if value == nil {
		return nil
	}
	return &provider.Record{ID: value.Record.ID, Platform: value.Record.Platform, Type: value.Record.Type, Credentials: value.Record.Credentials}
}
