package provider

import (
	"maps"
)

// ModelMappingDefaults 按需投影平台默认目录，不建立第二份别名缓存。
type ModelMappingDefaults struct {
	// Models 按提供商平台和认证类型提供当前默认模型目录。
	Models                func(*Record) []string
	Antigravity           func() map[string]string
	GoogleOne             func() map[string]string
	AntigravityAgentModel string
}

// ResolveModelMapping 读取不可变配置并返回独立映射，不在共享提供商内写入派生缓存。
func ResolveModelMapping(a *Record, defaults ModelMappingDefaults) map[string]string {
	rawMapping, _ := a.Credentials["model_mapping"].(map[string]any)
	if a.Credentials == nil {
		// Antigravity 平台使用默认映射
		if a.Platform == PlatformAntigravity {
			return maps.Clone(defaults.Antigravity())
		}
		// Bedrock 默认映射由 forwardBedrock 统一处理（需配合 region prefix 调整）
		return nil
	}
	if len(rawMapping) == 0 {
		if a.IsGeminiGoogleOne() {
			return maps.Clone(defaults.GoogleOne())
		}
		// Antigravity 平台使用默认映射
		if a.Platform == PlatformAntigravity {
			return maps.Clone(defaults.Antigravity())
		}
		return nil
	}

	result := make(map[string]string)
	for k, v := range rawMapping {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	if len(result) > 0 {
		return result
	}

	// Antigravity 平台使用默认映射
	if a.IsGeminiGoogleOne() {
		return maps.Clone(defaults.GoogleOne())
	}
	if a.Platform == PlatformAntigravity {
		return maps.Clone(defaults.Antigravity())
	}
	return nil
}
