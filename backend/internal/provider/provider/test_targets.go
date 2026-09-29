package provider

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// TestTargets 在一次原生提供商读取后选择固定的平台适配器，不建立缓存或第二套测试用例。
type TestTargets struct {
	Read        func(context.Context, int64) (*provider.Record, error)
	Qoder       *QoderProviderTest
	Gemini      *GeminiProviderTest
	Grok        *GrokProviderTest
	Anthropic   *AnthropicProviderTest
	OpenAI      *OpenAIProviderTest
	CN          *CNProviderTest
	Antigravity *AntigravityProviderTest
}

func (t *TestTargets) LoadTestTarget(ctx context.Context, request provider.TestRequest) (provider.TestTarget, error) {
	value, err := t.Read(ctx, request.ProviderID)
	if err != nil {
		return nil, err
	}
	switch value.Platform {
	case provider.PlatformQoder:
		return t.Qoder.Target(value), nil
	case provider.PlatformGemini:
		return t.Gemini.Target(value), nil
	case provider.PlatformGrok:
		return t.Grok.Target(value), nil
	case provider.PlatformOpenAI:
		return t.OpenAI.Target(value), nil
	case provider.PlatformKimi, provider.PlatformZhipu, provider.PlatformDeepseek, provider.PlatformMiniMax, provider.PlatformOpenCodeGo:
		return t.CN.Target(value), nil
	case provider.PlatformAntigravity:
		return t.Antigravity.Target(value), nil
	default:
		return t.Anthropic.Target(value), nil
	}
}
