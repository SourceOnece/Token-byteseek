package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
)

// NormalizeCodexModel 保留完整模型身份，只清理首尾空白。
func NormalizeCodexModel(model string) string {
	return openai.NormalizeCodexModel(model)
}
