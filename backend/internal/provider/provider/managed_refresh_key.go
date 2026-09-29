package provider

import "github.com/TokenFlux/TokenRouter/internal/provider"

// ManagedRefreshCacheKey 保持管理、后台及请求入口共享原有平台命名空间。
func ManagedRefreshCacheKey(value *provider.Record) string {
	if value.IsQoderCosy() {
		return provider.QoderTokenCacheKey(value)
	}
	switch value.Platform {
	case provider.PlatformOpenAI:
		return provider.OpenAITokenCacheKey(value)
	case provider.PlatformGemini:
		return GeminiTokenCacheKey(value)
	case provider.PlatformAntigravity:
		return provider.AntigravityTokenCacheKey(value)
	case provider.PlatformGrok:
		return provider.GrokTokenCacheKey(value)
	default:
		return provider.ClaudeTokenCacheKey(value)
	}
}
