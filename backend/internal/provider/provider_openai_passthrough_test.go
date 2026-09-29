package provider_test

import (
	"testing"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func TestProvider_IsOpenAIPassthroughEnabled(t *testing.T) {
	t.Run("新字段开启", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Extra: map[string]any{
				"openai_passthrough": true,
			},
		}
		require.True(t, provider.IsOpenAIPassthroughEnabled())
	})

	t.Run("兼容旧字段", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"openai_oauth_passthrough": true,
			},
		}
		require.True(t, provider.IsOpenAIPassthroughEnabled())
	})

	t.Run("非OpenAI提供商始终关闭", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformAnthropic,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"openai_passthrough": true,
			},
		}
		require.False(t, provider.IsOpenAIPassthroughEnabled())
	})

	t.Run("空额外配置默认关闭", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
		}
		require.False(t, provider.IsOpenAIPassthroughEnabled())
	})
}

func TestProvider_IsOpenAIOAuthPassthroughEnabled(t *testing.T) {
	t.Run("仅OAuth类型允许返回开启", func(t *testing.T) {
		oauthProvider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"openai_passthrough": true,
			},
		}
		require.True(t, oauthProvider.IsOpenAIOAuthPassthroughEnabled())

		apiKeyProvider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Extra: map[string]any{
				"openai_passthrough": true,
			},
		}
		require.False(t, apiKeyProvider.IsOpenAIOAuthPassthroughEnabled())
	})
}

func TestProvider_IsCodexCLIOnlyEnabled(t *testing.T) {
	t.Run("OpenAI OAuth 开启", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"codex_cli_only": true,
			},
		}
		require.True(t, provider.IsCodexCLIOnlyEnabled())
	})

	t.Run("OpenAI OAuth 关闭", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"codex_cli_only": false,
			},
		}
		require.False(t, provider.IsCodexCLIOnlyEnabled())
	})

	t.Run("字段缺失默认关闭", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra:    map[string]any{},
		}
		require.False(t, provider.IsCodexCLIOnlyEnabled())
	})

	t.Run("类型非法默认关闭", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"codex_cli_only": "true",
			},
		}
		require.False(t, provider.IsCodexCLIOnlyEnabled())
	})

	t.Run("非 OAuth 提供商始终关闭", func(t *testing.T) {
		apiKeyProvider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Extra: map[string]any{
				"codex_cli_only": true,
			},
		}
		require.False(t, apiKeyProvider.IsCodexCLIOnlyEnabled())

		otherPlatform := &providercore.Record{
			Platform: capability.PlatformAnthropic,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"codex_cli_only": true,
			},
		}
		require.False(t, otherPlatform.IsCodexCLIOnlyEnabled())
	})

	t.Run("新策略字段优先于旧字段", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"openai_oauth_client_policy": providercore.OpenAIOAuthClientPolicyAny,
				"codex_cli_only":             true,
			},
		}
		require.False(t, provider.IsCodexCLIOnlyEnabled())
		require.Equal(t, providercore.OpenAIOAuthClientPolicyAny, provider.GetOpenAIOAuthClientPolicy())
	})

	t.Run("TLS 路由器策略不等同于 Codex-only", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"openai_oauth_client_policy": providercore.OpenAIOAuthClientPolicyTLSRouterMatchedOnly,
				"tls_fingerprint_router_id":  int64(12),
			},
		}
		require.False(t, provider.IsCodexCLIOnlyEnabled())
		require.True(t, provider.IsOpenAIOAuthTLSRouterMatchedOnly())
		require.Equal(t, int64(12), provider.GetTLSFingerprintRouterID())
	})
}

func TestProvider_IsTLSFingerprintEnabled(t *testing.T) {
	tests := []struct {
		name     string
		provider *providercore.Record
		want     bool
	}{
		{
			name: "Anthropic OAuth 开启",
			provider: &providercore.Record{
				Platform: capability.PlatformAnthropic,
				Type:     capability.ProviderTypeOAuth,
				Extra:    map[string]any{"enable_tls_fingerprint": true},
			},
			want: true,
		},
		{
			name: "Anthropic SetupToken 开启",
			provider: &providercore.Record{
				Platform: capability.PlatformAnthropic,
				Type:     capability.ProviderTypeSetupToken,
				Extra:    map[string]any{"enable_tls_fingerprint": true},
			},
			want: true,
		},
		{
			name: "OpenAI OAuth 开启",
			provider: &providercore.Record{
				Platform: capability.PlatformOpenAI,
				Type:     capability.ProviderTypeOAuth,
				Extra:    map[string]any{"enable_tls_fingerprint": true},
			},
			want: true,
		},
		{
			name: "OpenAI API Key 不支持",
			provider: &providercore.Record{
				Platform: capability.PlatformOpenAI,
				Type:     capability.ProviderTypeAPIKey,
				Extra:    map[string]any{"enable_tls_fingerprint": true},
			},
			want: false,
		},
		{
			name: "非法类型按关闭处理",
			provider: &providercore.Record{
				Platform: capability.PlatformOpenAI,
				Type:     capability.ProviderTypeOAuth,
				Extra:    map[string]any{"enable_tls_fingerprint": "true"},
			},
			want: false,
		},
		{
			name: "字段缺失按关闭处理",
			provider: &providercore.Record{
				Platform: capability.PlatformOpenAI,
				Type:     capability.ProviderTypeOAuth,
				Extra:    map[string]any{},
			},
			want: false,
		},
		{
			name:     "nil 提供商按关闭处理",
			provider: nil,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.provider.IsTLSFingerprintEnabled())
		})
	}
}

func TestProvider_IsOpenAIResponsesWebSocketV2Enabled(t *testing.T) {
	t.Run("OAuth使用OAuth专用开关", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"openai_oauth_responses_websockets_v2_enabled": true,
			},
		}
		require.True(t, provider.IsOpenAIResponsesWebSocketV2Enabled())
	})

	t.Run("API Key使用API Key专用开关", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
			},
		}
		require.True(t, provider.IsOpenAIResponsesWebSocketV2Enabled())
	})

	t.Run("OAuth提供商不会读取API Key专用开关", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": true,
			},
		}
		require.False(t, provider.IsOpenAIResponsesWebSocketV2Enabled())
	})

	t.Run("分类型新键优先于兼容键", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"openai_oauth_responses_websockets_v2_enabled": false,
				"responses_websockets_v2_enabled":              true,
				"openai_ws_enabled":                            true,
			},
		}
		require.False(t, provider.IsOpenAIResponsesWebSocketV2Enabled())
	})

	t.Run("分类型键缺失时回退兼容键", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Extra: map[string]any{
				"responses_websockets_v2_enabled": true,
			},
		}
		require.True(t, provider.IsOpenAIResponsesWebSocketV2Enabled())
	})

	t.Run("非OpenAI提供商默认关闭", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformAnthropic,
			Type:     capability.ProviderTypeAPIKey,
			Extra: map[string]any{
				"responses_websockets_v2_enabled": true,
			},
		}
		require.False(t, provider.IsOpenAIResponsesWebSocketV2Enabled())
	})
}

func TestProvider_ResolveOpenAIResponsesWebSocketV2Mode(t *testing.T) {
	t.Run("default fallback to ctx_pool", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra:    map[string]any{},
		}
		require.Equal(t, providercore.OpenAIWSIngressModeCtxPool, provider.ResolveOpenAIResponsesWebSocketV2Mode(""))
		require.Equal(t, providercore.OpenAIWSIngressModeCtxPool, provider.ResolveOpenAIResponsesWebSocketV2Mode("invalid"))
	})

	t.Run("oauth mode field has highest priority", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"openai_oauth_responses_websockets_v2_mode":    providercore.OpenAIWSIngressModePassthrough,
				"openai_oauth_responses_websockets_v2_enabled": false,
				"responses_websockets_v2_enabled":              false,
			},
		}
		require.Equal(t, providercore.OpenAIWSIngressModePassthrough, provider.ResolveOpenAIResponsesWebSocketV2Mode(providercore.OpenAIWSIngressModeCtxPool))
	})

	t.Run("legacy enabled maps to ctx_pool", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Extra: map[string]any{
				"responses_websockets_v2_enabled": true,
			},
		}
		require.Equal(t, providercore.OpenAIWSIngressModeCtxPool, provider.ResolveOpenAIResponsesWebSocketV2Mode(providercore.OpenAIWSIngressModeOff))
	})

	t.Run("shared/dedicated mode strings are compatible with ctx_pool", func(t *testing.T) {
		shared := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"openai_oauth_responses_websockets_v2_mode": providercore.OpenAIWSIngressModeShared,
			},
		}
		dedicated := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"openai_oauth_responses_websockets_v2_mode": providercore.OpenAIWSIngressModeDedicated,
			},
		}
		require.Equal(t, providercore.OpenAIWSIngressModeShared, shared.ResolveOpenAIResponsesWebSocketV2Mode(providercore.OpenAIWSIngressModeOff))
		require.Equal(t, providercore.OpenAIWSIngressModeDedicated, dedicated.ResolveOpenAIResponsesWebSocketV2Mode(providercore.OpenAIWSIngressModeOff))
		require.Equal(t, providercore.OpenAIWSIngressModeCtxPool, providercore.NormalizeOpenAIWSIngressDefaultMode(providercore.OpenAIWSIngressModeShared))
		require.Equal(t, providercore.OpenAIWSIngressModeCtxPool, providercore.NormalizeOpenAIWSIngressDefaultMode(providercore.OpenAIWSIngressModeDedicated))
	})

	t.Run("legacy disabled maps to off", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Extra: map[string]any{
				"openai_apikey_responses_websockets_v2_enabled": false,
				"responses_websockets_v2_enabled":               true,
			},
		}
		require.Equal(t, providercore.OpenAIWSIngressModeOff, provider.ResolveOpenAIResponsesWebSocketV2Mode(providercore.OpenAIWSIngressModeCtxPool))
	})

	t.Run("non openai always off", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformAnthropic,
			Type:     capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"openai_oauth_responses_websockets_v2_mode": providercore.OpenAIWSIngressModeDedicated,
			},
		}
		require.Equal(t, providercore.OpenAIWSIngressModeOff, provider.ResolveOpenAIResponsesWebSocketV2Mode(providercore.OpenAIWSIngressModeDedicated))
	})
}

func TestProvider_OpenAIWSExtraFlags(t *testing.T) {
	provider := &providercore.Record{
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeOAuth,
		Extra: map[string]any{
			"openai_ws_force_http":           true,
			"openai_ws_allow_store_recovery": true,
		},
	}
	require.True(t, provider.IsOpenAIWSForceHTTPEnabled())
	require.True(t, provider.IsOpenAIWSAllowStoreRecoveryEnabled())

	off := &providercore.Record{Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Extra: map[string]any{}}
	require.False(t, off.IsOpenAIWSForceHTTPEnabled())
	require.False(t, off.IsOpenAIWSAllowStoreRecoveryEnabled())

	var nilProvider *providercore.Record
	require.False(t, nilProvider.IsOpenAIWSAllowStoreRecoveryEnabled())

	nonOpenAI := &providercore.Record{
		Platform: capability.PlatformAnthropic,
		Type:     capability.ProviderTypeOAuth,
		Extra: map[string]any{
			"openai_ws_allow_store_recovery": true,
		},
	}
	require.False(t, nonOpenAI.IsOpenAIWSAllowStoreRecoveryEnabled())
}
