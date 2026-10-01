package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 显式白名单与完整模型语义一致，不能靠后缀或旧拼写取得基础型号权限。
func TestGroupAllowlistUsesCompleteIdentity(t *testing.T) {
	allow := GroupModelAllowlist{Enabled: true, Models: []string{"gpt-6.1-sol", "claude-opus-4-6", "gemini-3.8-flash"}}
	for _, model := range []string{"gpt-6.1-sol", "models/gemini-3.8-flash"} {
		require.True(t, allow.Allows(model), model)
	}
	for _, model := range []string{"gpt-6.1-sol-max", "claude-opus-4.6", "claude-opus-4-6-thinking"} {
		require.False(t, allow.Allows(model), model)
	}
	allow.Models = []string{"gpt-6.1-*"}
	require.True(t, allow.Allows("gpt-6.1-sol"))
}
