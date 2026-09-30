package openai

import (
	"fmt"
	"strings"
)

// IsGPT61SolModel 仅识别已发布的 Sol 型号和本地推理/压缩后缀，不匹配未知变体。
func IsGPT61SolModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	if index := strings.LastIndexByte(model, '/'); index >= 0 {
		model = model[index+1:]
	}
	model = strings.ReplaceAll(model, "_", "-")
	if model == "gpt-6.1-sol" {
		return true
	}
	suffix, ok := strings.CutPrefix(model, "gpt-6.1-sol-")
	if !ok {
		return false
	}
	switch suffix {
	case "none", "minimal", "low", "medium", "high", "xhigh", "max", "openai-compact":
		return true
	}
	return false
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
