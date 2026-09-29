package provider

// AllowedSchedulingThresholdPlatforms 是允许设置提供商自动停调阈值的平台列表。
// openai/anthropic/grok 有原生用量窗口；kimi/zhipu 的 Coding Plan 同样暴露 5h/weekly
// 滚动窗口，纳入阈值评估。deepseek 为余额型，走余额检测而非阈值。
var AllowedSchedulingThresholdPlatforms = []string{
	PlatformOpenAI,
	PlatformAnthropic,
	PlatformGrok,
	PlatformKimi,
	PlatformZhipu,
}

const AnthropicFableRateLimitKey = "claude-fable-5"
