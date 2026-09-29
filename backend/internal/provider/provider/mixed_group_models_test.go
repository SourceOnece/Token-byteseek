package provider

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/anthropic"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/stretchr/testify/require"
)

func TestMixedGroupProviderModelScope(t *testing.T) {
	// 同组选号前必须识别提供商的实际模型范围，不能把空配置当作全模型。
	claude := anthropic.DefaultModelIDs()[0]
	gpt := openai.DefaultModelIDs()[0]
	for _, tc := range []struct {
		platform string
		own      string
		other    string
	}{
		{provider.PlatformAnthropic, claude, gpt},
		{provider.PlatformOpenAI, gpt, claude},
	} {
		t.Run(tc.platform, func(t *testing.T) {
			value := &provider.Record{Platform: tc.platform, Type: provider.ProviderTypeAPIKey}
			require.True(t, value.IsModelSupported(tc.own, ModelDefaults(), ModelRules(value)))
			require.False(t, value.IsModelSupported(tc.other, ModelDefaults(), ModelRules(value)))
			require.False(t, value.IsModelSupported("unknown-model", ModelDefaults(), ModelRules(value)))
			value.Credentials = map[string]any{"model_mapping": map[string]any{"custom-alias": "custom-upstream"}}
			require.True(t, value.IsModelSupported("custom-alias", ModelDefaults(), ModelRules(value)))
			value.Credentials["model_whitelist"] = []string{"custom-*"}
			require.True(t, value.IsModelSupported("custom-model", ModelDefaults(), ModelRules(value)))
			require.False(t, value.IsModelSupported(tc.own, ModelDefaults(), ModelRules(value)))
			require.NotContains(t, value.GetConfiguredRequestModels(ModelDefaults()), "custom-*")
			value.Credentials["model_whitelist"] = []string{"*"}
			require.True(t, value.IsModelSupported(tc.other, ModelDefaults(), ModelRules(value)))
			require.NotContains(t, value.GetConfiguredRequestModels(ModelDefaults()), "*")
		})
	}
}
