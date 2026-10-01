package openai

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// IsGPT61SolModel 按完整型号识别能力，不恢复已经退役的隐式 effort 后缀。
func IsGPT61SolModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	model = strings.TrimPrefix(model, "openai/")
	return model == "gpt-6.1-sol"
}

// ValidateGPT61SolEffort 拒绝不支持的关闭推理请求，不能静默升档。
func ValidateGPT61SolEffort(model, effort string) error {
	if IsGPT61SolModel(model) {
		switch strings.ToLower(strings.TrimSpace(effort)) {
		case "none", "minimal":
			return fmt.Errorf("gpt-6.1-sol does not support reasoning effort %q; use low, medium, high, xhigh or max", effort)
		}
	}
	return nil
}

// ValidateGPT61CompatBody 在最终模型映射后、转换前检查原始显式参数。
func ValidateGPT61CompatBody(body []byte, model string, chatOnly bool) error {
	if !IsGPT61SolModel(model) {
		return nil
	}
	for _, path := range []string{"reasoning.effort", "reasoning_effort", "output_config.effort"} {
		if err := ValidateGPT61SolEffort(model, gjson.GetBytes(body, path).String()); err != nil {
			return err
		}
	}
	if gjson.GetBytes(body, "thinking.type").String() == "disabled" {
		return ValidateGPT61SolEffort(model, "none")
	}
	if chatOnly && (len(gjson.GetBytes(body, "tools").Array()) > 0 || len(gjson.GetBytes(body, "functions").Array()) > 0) {
		return fmt.Errorf("gpt-6.1-sol requires Responses for tool calls; this provider only supports Chat Completions")
	}
	return nil
}
