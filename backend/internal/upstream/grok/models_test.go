package grok

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNativeModelCatalogue 验证默认模型仅自映射，默认文本设置不会增加隐式别名。
func TestNativeModelCatalogue(t *testing.T) {
	mapping := ModelMappingWithOptions(ModelMappingOptions{DefaultText: "custom-grok"})
	require.NotEmpty(t, mapping)
	for from, to := range mapping {
		require.Equal(t, from, to)
	}
	for _, alias := range []string{"grok", "grok-latest", "grok-build", "gpt-*", "claude-*", "xai/grok", "grok-imagine-edit", "grok-imagine-video-1.5-preview"} {
		require.NotContains(t, mapping, alias)
	}
	require.Contains(t, mapping, "grok-4.6")
	require.Contains(t, mapping, DefaultImagineVideo15Model)
}

// TestExplicitGrokIDsRemainUnchanged 验证文本、媒体和供应商限定名均保留原值。
func TestExplicitGrokIDsRemainUnchanged(t *testing.T) {
	for _, model := range []string{"grok", "grok-latest", "grok-4.6-latest", "grok-build-latest", "grok-4.20-multi-agent", "xai/grok-4.6", "grok-imagine-video-1.5-preview"} {
		require.Equal(t, model, NormalizeModelID(model))
		require.Equal(t, model, ResolveGrokTextResponsesModelID(model, "custom-default"))
		require.Equal(t, model, CanonicalImagineVideoModel(model))
	}
	require.Equal(t, "custom-default", ResolveGrokTextResponsesModelID("", "custom-default"))
	require.True(t, IsGrokTextResponsesModelID("grok-4.6"))
	require.False(t, IsGrokTextResponsesModelID("grok-latest"))
}
