package anthropic

import (
	"fmt"
	"github.com/tidwall/gjson"
)

// ValidateClaude55Request 在请求改写前拒绝模型不支持的组合，保持错误可解释。
func ValidateClaude55Request(body []byte, model string) error {
	if !IsOpus55(model) && !IsSonnet55(model) {
		return nil
	}
	isSonnet55 := IsSonnet55(model)
	switch gjson.GetBytes(body, "thinking.type").String() {
	case "disabled", "enabled":
		if isSonnet55 {
			return fmt.Errorf("claude-sonnet-5-5 requires adaptive thinking or thinking.type=between_tools; omit thinking or use one of those modes")
		}
		return fmt.Errorf("claude-opus-5-5 requires adaptive thinking; omit thinking or use thinking.type=adaptive and output_config.effort")
	case "between_tools":
		if !isSonnet55 {
			return fmt.Errorf("claude-opus-5-5 requires adaptive thinking; thinking.type=between_tools is unsupported")
		}
		effort := gjson.GetBytes(body, "output_config.effort").String()
		if effort == "xhigh" || effort == "max" {
			return fmt.Errorf("claude-sonnet-5-5 thinking.type=between_tools supports only low, medium or high effort")
		}
		for _, field := range []string{"thinking.display", "thinking.budget_tokens", "thinking.block_binding"} {
			if gjson.GetBytes(body, field).Exists() {
				return fmt.Errorf("claude-sonnet-5-5 thinking.type=between_tools does not support %s", field)
			}
		}
	}
	modelName := "claude-opus-5-5"
	if isSonnet55 {
		modelName = "claude-sonnet-5-5"
	}
	if gjson.GetBytes(body, "tool_choice").String() == "required" {
		return fmt.Errorf("%s does not support forced tool_choice; use auto or none", modelName)
	}
	switch gjson.GetBytes(body, "tool_choice.type").String() {
	case "any", "tool", "function", "custom", "namespace":
		return fmt.Errorf("%s does not support forced tool_choice; use auto or none", modelName)
	}
	if isSonnet55 {
		if temperature := gjson.GetBytes(body, "temperature"); temperature.Exists() && (temperature.Type != gjson.Number || temperature.Float() != 1) {
			return fmt.Errorf("claude-sonnet-5-5 does not support non-default temperature")
		}
		if topP := gjson.GetBytes(body, "top_p"); topP.Exists() && (topP.Type != gjson.Number || topP.Float() < 0.99 || topP.Float() > 1) {
			return fmt.Errorf("claude-sonnet-5-5 does not support non-default top_p")
		}
		if gjson.GetBytes(body, "top_k").Exists() {
			return fmt.Errorf("claude-sonnet-5-5 does not support top_k")
		}
	}
	return nil
}
