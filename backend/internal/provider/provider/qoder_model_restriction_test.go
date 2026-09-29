package provider

import (
	"testing"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func TestProviderIsModelSupported_QoderMappingWhitelistSemantics(t *testing.T) {
	tests := []struct {
		name           string
		credentials    map[string]any
		requestedModel string
		expected       bool
	}{
		{
			name: "mapping only does not restrict unmatched request model",
			credentials: map[string]any{
				"model_mapping": map[string]any{
					"claude-opus-4-6": "ultimate",
				},
			},
			requestedModel: "auto",
			expected:       true,
		},
		{
			name: "explicit empty whitelist keeps mapping unrestricted",
			credentials: map[string]any{
				"model_mapping": map[string]any{
					"claude-opus-4-6": "ultimate",
				},
				"model_whitelist": []any{},
			},
			requestedModel: "auto",
			expected:       true,
		},
		{
			name: "whitelist allows mapped final route key",
			credentials: map[string]any{
				"model_mapping": map[string]any{
					"claude-opus-4-6": "ultimate",
				},
				"model_whitelist": []any{"ultimate"},
			},
			requestedModel: "claude-opus-4-6",
			expected:       true,
		},
		{
			name: "whitelist rejects final model miss",
			credentials: map[string]any{
				"model_mapping": map[string]any{
					"claude-opus-4-6": "ultimate",
				},
				"model_whitelist": []any{"ultimate"},
			},
			requestedModel: "auto",
			expected:       false,
		},
		{
			name: "whitelist public alias matches raw route key",
			credentials: map[string]any{
				"model_whitelist": []any{"claude-opus-4-6"},
			},
			requestedModel: "ultimate",
			expected:       true,
		},
		{
			name: "legacy raw self mapping is not treated as a whitelist",
			credentials: map[string]any{
				"model_mapping": map[string]any{
					"ultimate": "ultimate",
				},
			},
			requestedModel: "auto",
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &providercore.Record{
				Platform:    capability.PlatformQoder,
				Credentials: tt.credentials,
			}

			require.Equal(t, tt.expected, provider.IsModelSupported(tt.requestedModel, ModelDefaults(), ModelRules(provider)))
		})
	}
}

func TestProviderGetConfiguredRequestModels_QoderMappingWhitelistSemantics(t *testing.T) {
	mappingOnly := &providercore.Record{
		Platform: capability.PlatformQoder,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"claude-opus-4-6": "ultimate",
			},
		},
	}
	models := mappingOnly.GetConfiguredRequestModels(ModelDefaults())
	require.Contains(t, models, "claude-opus-4-6")
	require.Contains(t, models, "auto")
	require.Contains(t, models, "ultimate")

	withWhitelist := &providercore.Record{
		Platform: capability.PlatformQoder,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"claude-opus-4-6": "ultimate",
			},
			"model_whitelist": []any{"ultimate"},
		},
	}
	require.ElementsMatch(t, []string{"claude-opus-4-6", "ultimate"}, withWhitelist.GetConfiguredRequestModels(ModelDefaults()))

	whitelistOnly := &providercore.Record{
		Platform: capability.PlatformQoder,
		Credentials: map[string]any{
			"model_whitelist": []any{"claude-opus-4-6", "glm-5.2"},
		},
	}
	require.Equal(t, []string{"claude-opus-4-6", "glm-5.2"}, whitelistOnly.GetConfiguredRequestModels(ModelDefaults()))
}

func TestProviderIsModelSupported_QoderSiteCompatibility(t *testing.T) {
	global := &providercore.Record{Platform: capability.PlatformQoder, Credentials: map[string]any{"site": "global", "model_whitelist": []string{"*"}}}
	cn := &providercore.Record{Platform: capability.PlatformQoder, Credentials: map[string]any{"site": "cn", "model_whitelist": []string{"*"}}}

	require.True(t, global.IsModelSupported("claude-opus-4-6", ModelDefaults(), ModelRules(global)))
	require.False(t, cn.IsModelSupported("claude-opus-4-6", ModelDefaults(), ModelRules(cn)))
	require.False(t, global.IsModelSupported("qwen3.6-flash", ModelDefaults(), ModelRules(global)))
	require.True(t, cn.IsModelSupported("qwen3.6-flash", ModelDefaults(), ModelRules(cn)))
	require.False(t, global.IsModelSupported("q36fmodel", ModelDefaults(), ModelRules(global)))
	require.True(t, cn.IsModelSupported("q36fmodel", ModelDefaults(), ModelRules(cn)))
	require.True(t, global.IsModelSupported("mmodel", ModelDefaults(), ModelRules(global)))
	require.True(t, cn.IsModelSupported("mmodel", ModelDefaults(), ModelRules(cn)))
	require.True(t, global.IsModelSupported("unknown-raw-key", ModelDefaults(), ModelRules(global)))

	cn.Credentials["model_mapping"] = map[string]any{"claude-opus-4-6": "ultimate"}
	require.False(t, cn.IsModelSupported("claude-opus-4-6", ModelDefaults(), ModelRules(cn)), "显式映射不能绕过站点的上游模型限制")
	cn.Credentials["model_mapping"] = map[string]any{"claude-opus-4-6": "q36fmodel"}
	require.True(t, cn.IsModelSupported("claude-opus-4-6", ModelDefaults(), ModelRules(cn)), "别名可以映射到该站点支持的模型")
}
