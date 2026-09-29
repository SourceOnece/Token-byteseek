package httpapi

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/gateway/modeltrace"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
)

// ResolveOpenAIMessagesProviderLayerModel 将通用分组映射结果规范化后交给提供商模型规则。
func ResolveOpenAIMessagesProviderLayerModel(groupMappedModel string) string {
	return gatewayadapter.NormalizeOpenAICompatRequestedModel(groupMappedModel)
}

// ResolveOpenAIMessagesProviderLayerModelForRequest 登记规范化结果，供响应恢复和用量追踪使用。
func ResolveOpenAIMessagesProviderLayerModelForRequest(ctx context.Context, groupMappedModel string) string {
	model := ResolveOpenAIMessagesProviderLayerModel(groupMappedModel)
	modeltrace.RegisterStage(ctx, model)
	return model
}
