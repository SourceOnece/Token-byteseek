package provider_test

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/protocol/openai"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/stretchr/testify/require"
)

func TestShouldApplyChatGPTAccountInfoPlanType(t *testing.T) {
	require.False(t, provider.ShouldApplyChatGPTAccountInfoPlanType("pro", "self_serve_business_usage_based"))
	require.False(t, provider.ShouldApplyChatGPTAccountInfoPlanType("free", "team"))
	require.False(t, provider.ShouldApplyChatGPTAccountInfoPlanType("", ""))
	require.True(t, provider.ShouldApplyChatGPTAccountInfoPlanType("", "pro"))
}

func TestChatGPTAccountInfoBelongsToTokenProvider(t *testing.T) {
	require.False(t, provider.ChatGPTAccountInfoBelongsToTokenProvider(
		&provider.OpenAITokenInfo{ChatGPTAccountID: "personal-a"},
		&openai.ChatGPTAccountInfo{ProviderID: "workspace-b"},
	))
	require.True(t, provider.ChatGPTAccountInfoBelongsToTokenProvider(
		&provider.OpenAITokenInfo{ChatGPTAccountID: "personal-a"},
		&openai.ChatGPTAccountInfo{ProviderID: "PERSONAL-A"},
	))
	// 任一侧缺少 ID 时无法区分，保持既有行为。
	require.True(t, provider.ChatGPTAccountInfoBelongsToTokenProvider(
		&provider.OpenAITokenInfo{},
		&openai.ChatGPTAccountInfo{ProviderID: "workspace-b"},
	))
	require.True(t, provider.ChatGPTAccountInfoBelongsToTokenProvider(
		&provider.OpenAITokenInfo{ChatGPTAccountID: "personal-a"},
		&openai.ChatGPTAccountInfo{},
	))
}
