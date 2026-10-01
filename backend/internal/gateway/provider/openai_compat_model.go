package provider

import (
	"strings"

	protocolanthropic "github.com/TokenFlux/TokenRouter/internal/protocol/anthropic"
	protocolopenai "github.com/TokenFlux/TokenRouter/internal/protocol/openai"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
)

// OpenAICompatAnthropicReasoningEffort 在最终上游模型确定后重新裁定 Messages 桥接的推理强度。
// Anthropic 的 max 通常转换为 OpenAI xhigh，但 GPT-5.6 支持原生 max，不能因客户端别名而降级。
func OpenAICompatAnthropicReasoningEffort(req *protocolanthropic.AnthropicRequest, upstreamModel, convertedEffort string) string {
	if req == nil || req.OutputConfig == nil || !strings.EqualFold(strings.TrimSpace(req.OutputConfig.Effort), "max") {
		return convertedEffort
	}
	if protocolopenai.IsGPT61SolModel(upstreamModel) {
		return "max"
	}
	if normalized := capability.NormalizeRecordedOpenAIEffortForModel(req.OutputConfig.Effort, upstreamModel); normalized != "" {
		return normalized
	}
	if strings.EqualFold(strings.TrimSpace(convertedEffort), "max") {
		return "xhigh"
	}
	return convertedEffort
}
