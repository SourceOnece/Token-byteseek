package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
)

// NormalizeResponsesLiteForProvider 在调用者确认 Lite 标记后按原提供商资格选择唯一 wire 规范化。
func NormalizeResponsesLiteForProvider(value *provider.Record, body []byte) ([]byte, bool, error) {
	if value == nil || value.Platform != capability.PlatformOpenAI {
		return body, false, nil
	}
	if value.IsOpenAIOAuth() {
		return openai.NormalizeResponsesLiteToolsPayload(body)
	}
	return openai.NormalizeResponsesLiteParallelToolCallsPayload(body)
}
