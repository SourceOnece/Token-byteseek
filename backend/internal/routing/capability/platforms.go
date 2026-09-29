package capability

// ProviderPlatforms 返回提供商平台目录的独立副本，供跨平台分组构造候选池。
func ProviderPlatforms() []string {
	return []string{PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformAntigravity, PlatformGrok, PlatformQoder, PlatformKimi, PlatformZhipu, PlatformDeepseek, PlatformMiniMax, PlatformOpenCodeGo}
}
