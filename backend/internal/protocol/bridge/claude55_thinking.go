package bridge

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	claudewire "github.com/TokenFlux/TokenRouter/internal/protocol/anthropic"
	"strings"
)

const anthropicThinkingEnvelopePrefix = "anthropic-thinking-v1:"

// 签名封装与 sub2api 相同，只保留原始上游思考块，不生成或伪造签名。
func encodeAnthropicThinking(block AnthropicContentBlock) string {
	payload, err := json.Marshal(struct {
		Type      string `json:"type"`
		Thinking  string `json:"thinking"`
		Signature string `json:"signature,omitempty"`
		Data      string `json:"data,omitempty"`
	}{block.Type, block.Thinking, block.Signature, block.Data})
	if err != nil {
		return ""
	}
	return anthropicThinkingEnvelopePrefix + base64.RawStdEncoding.EncodeToString(payload)
}

func decodeAnthropicThinking(value string) (AnthropicContentBlock, bool) {
	var block AnthropicContentBlock
	if !strings.HasPrefix(value, anthropicThinkingEnvelopePrefix) {
		return block, false
	}
	data, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, anthropicThinkingEnvelopePrefix))
	if err != nil || json.Unmarshal(data, &block) != nil {
		return AnthropicContentBlock{}, false
	}
	return block, (block.Type == "thinking" && block.Signature != "") || (block.Type == "redacted_thinking" && block.Data != "")
}

// 5.5 使用自适应推理，不能套用旧版的固定预算与强制工具选择。
func applyClaude55Reasoning(req *ResponsesRequest, out *AnthropicRequest) error {
	var choice struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(out.ToolChoice, &choice)
	if choice.Type == "any" || choice.Type == "tool" {
		return fmt.Errorf("Claude 5.5 does not support forced tool_choice") //nolint:staticcheck // 保持客户端可读的协议错误首字母。
	}
	sonnet := claudewire.IsSonnet55(req.Model)
	effort := "medium"
	if sonnet {
		effort = "high"
		if req.Temperature != nil && *req.Temperature != 1 {
			return fmt.Errorf("claude-sonnet-5-5 does not support non-default temperature")
		}
		if req.TopP != nil && (*req.TopP < 0.99 || *req.TopP > 1) {
			return fmt.Errorf("claude-sonnet-5-5 does not support non-default top_p")
		}
	}
	if req.Reasoning != nil && strings.TrimSpace(req.Reasoning.Effort) != "" {
		effort = strings.ToLower(strings.TrimSpace(req.Reasoning.Effort))
	}
	if sonnet && effort == "none" {
		out.Thinking = &AnthropicThinking{Type: "between_tools"}
		out.OutputConfig = &AnthropicOutputConfig{Effort: "low"}
		return nil
	}
	switch effort {
	case "low", "medium", "high", "xhigh", "max":
	default:
		return fmt.Errorf("reasoning effort %q is not supported by Claude 5.5", effort)
	}
	out.Thinking = &AnthropicThinking{Type: "adaptive"}
	out.OutputConfig = &AnthropicOutputConfig{Effort: effort}
	return nil
}
