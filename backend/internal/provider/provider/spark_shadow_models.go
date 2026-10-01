package provider

import "github.com/TokenFlux/TokenRouter/internal/upstream/openai"

// sparkModelVariants 返回 Spark 的完整原生型号。
func sparkModelVariants() []string {
	out := make([]string, 0, 1)
	for alias, target := range openai.CodexModelMap {
		if target == "gpt-5.3-codex-spark" {
			out = append(out, alias)
		}
	}
	return out
}

// DefaultSparkShadowModels 返回 spark 影子提供商的默认 model_mapping。
//
// 恒等映射（key 映射到自身）把「只接 spark」限制落在 key 白名单上，模型零改写、
// 与空 mapping 透传行为一致。推理强度通过独立请求字段传递。
func DefaultSparkShadowModels() map[string]any {
	variants := sparkModelVariants()
	mapping := make(map[string]any, len(variants))
	for _, m := range variants {
		mapping[m] = m
	}
	return mapping
}
