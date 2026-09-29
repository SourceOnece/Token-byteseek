package provider

import (
	"maps"
)

// DiscardDeprecatedProviderExtra 静默移除旧客户端可能继续提交的废弃提供商扩展键。
func DiscardDeprecatedProviderExtra(extra map[string]any) {
	NormalizeLegacyOpenAIProviderExtra(extra)
	DiscardDeprecatedExtra(extra)
}

// NormalizeDeprecatedProviderExtraUpdate 规范化整份替换语义的提供商 Extra 更新。
// 第二个返回值表示是否仍应执行替换：显式空对象保留清空语义，只有废弃键的对象视为未提供更新。
func NormalizeDeprecatedProviderExtraUpdate(extra map[string]any) (map[string]any, bool) {
	if extra == nil {
		return nil, false
	}
	normalized := maps.Clone(extra)
	DiscardDeprecatedProviderExtra(normalized)
	if len(extra) > 0 && len(normalized) == 0 {
		return nil, false
	}
	return normalized, true
}
