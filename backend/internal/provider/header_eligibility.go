package provider

const credKeyHeaderOverrideEnabled = "header_override_enabled"

// IsHeaderOverrideEligible 报告提供商类型是否支持请求头覆写。
// Anthropic / OpenAI / 国产供应商仅开放 api_key 提供商；Grok 额外开放 oauth 提供商——
// 订阅流量改发自定义转发地址时，通常需要补充中间层要求的准入头。
func (a *Record) IsHeaderOverrideEligible() bool {
	if a == nil {
		return false
	}
	switch a.Platform {
	case PlatformAnthropic, PlatformOpenAI, PlatformKimi, PlatformZhipu, PlatformDeepseek, PlatformMiniMax, PlatformOpenCodeGo:
		return a.Type == ProviderTypeAPIKey
	case PlatformGrok:
		return a.Type == ProviderTypeAPIKey || a.Type == ProviderTypeOAuth
	default:
		return false
	}
}

// IsHeaderOverrideEnabled 报告提供商是否启用了请求头覆写。
func (a *Record) IsHeaderOverrideEnabled() bool {
	if !a.IsHeaderOverrideEligible() || a.Credentials == nil {
		return false
	}
	enabled, ok := a.Credentials[credKeyHeaderOverrideEnabled].(bool)
	return ok && enabled
}
