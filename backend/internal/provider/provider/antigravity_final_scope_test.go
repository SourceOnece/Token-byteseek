package provider

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/stretchr/testify/require"
)

// thinking 变换只改变最终名称，不重新读取映射表跳到另一个模型。
func TestAntigravityFinalModelChecksWhitelistAfterThinking(t *testing.T) {
	enabled := true
	disabled := false
	value := &provider.Record{Platform: provider.PlatformAntigravity, Type: provider.ProviderTypeOAuth, Credentials: map[string]any{
		"model_mapping":   map[string]any{"public": "claude-sonnet-4-5", "claude-sonnet-4-5-thinking": "must-not-map-again"},
		"model_whitelist": []string{"claude-sonnet-4-5-thinking"},
	}}
	require.Equal(t, "claude-sonnet-4-5-thinking", FinalAntigravityModel(value, "public", &enabled))
	require.Empty(t, FinalAntigravityModel(value, "public", &disabled))
	require.Empty(t, MapAntigravityModel(value, "public"))
	value.Credentials["model_whitelist"] = []string{"claude-sonnet-4-5"}
	require.Empty(t, FinalAntigravityModel(value, "public", &enabled))
	require.Equal(t, "claude-sonnet-4-5", MapAntigravityModel(value, "public"))
}

func TestAntigravityMappedTargetCannotBypassExplicitWhitelist(t *testing.T) {
	value := &provider.Record{Platform: provider.PlatformAntigravity, Credentials: map[string]any{
		"model_mapping":   map[string]any{"public": "custom-upstream"},
		"model_whitelist": []string{"different-model"},
	}}
	require.Empty(t, MapAntigravityModel(value, "public"))
	value.Credentials["model_whitelist"] = []string{"custom-*"}
	require.Equal(t, "custom-upstream", MapAntigravityModel(value, "public"))
}
