package provider

import (
	"testing"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func TestProviderSupportsOpenAIEndpointCapability(t *testing.T) {
	t.Run("OpenAI APIKey 默认兼容 chat、embeddings 和 alpha search", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
		}

		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityTextGeneration))
		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityEmbeddings))
		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityAlphaSearch))
	})

	t.Run("OpenAI OAuth 默认仅兼容 chat", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
		}

		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityTextGeneration))
		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityAlphaSearch))
		require.False(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityEmbeddings))
	})

	t.Run("alpha search 允许 OpenAI OAuth/PAT 与 APIKey 提供商，拒绝 Grok", func(t *testing.T) {
		// OAuth/PAT 走 chatgpt.com Codex 端点，APIKey 走 {base_url}/v1/alpha/search，
		// 两类都能承接独立搜索（APIKey 被排除曾导致纯 APIKey 分组搜索失效的回归）。
		apiKey := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
		}
		oauth := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
		}
		grok := &providercore.Record{
			Platform: capability.PlatformGrok,
			Type:     capability.ProviderTypeAPIKey,
		}

		require.True(t, SupportsOpenAIEndpoint(apiKey, providercore.OpenAIEndpointCapabilityAlphaSearch))
		require.True(t, SupportsOpenAIEndpoint(oauth, providercore.OpenAIEndpointCapabilityAlphaSearch))
		require.False(t, SupportsOpenAIEndpoint(grok, providercore.OpenAIEndpointCapabilityAlphaSearch))
	})

	t.Run("显式列表支持同时声明 chat 和 embeddings", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Credentials: map[string]any{
				"openai_workload_capabilities": []any{"text_generation", "embeddings"},
			},
		}

		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityTextGeneration))
		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityEmbeddings))
	})

	t.Run("显式列表只声明 chat 时不支持 embeddings", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Credentials: map[string]any{
				"openai_workload_capabilities": []any{"text_generation"},
			},
		}

		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityTextGeneration))
		// chat 能力隐含放行 alpha search（OAuth/APIKey 语义一致）。
		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityAlphaSearch))
		require.False(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityEmbeddings))
	})

	t.Run("OAuth 显式列表沿用 chat 能力放行 alpha search", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Credentials: map[string]any{
				"openai_workload_capabilities": []any{"text_generation"},
			},
		}

		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityAlphaSearch))
	})

	// OAuth 历史数据可能保留空能力容器，应与缺失字段一样不阻断文本调度。
	for _, emptyCapabilities := range []struct {
		name  string
		value any
	}{
		{name: "map[string]any", value: map[string]any{}},
		{name: "[]any", value: []any{}},
		{name: "[]string", value: []string{}},
	} {
		t.Run("OAuth 空能力 "+emptyCapabilities.name, func(t *testing.T) {
			provider := &providercore.Record{
				Platform: capability.PlatformOpenAI,
				Type:     capability.ProviderTypeOAuth,
				Credentials: map[string]any{
					providercore.OpenAIWorkloadCapabilitiesCredentialKey: emptyCapabilities.value,
				},
			}

			require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityTextGeneration))
			require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityResponses))
		})
	}

	t.Run("API Key 空能力仍表示显式禁用", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Credentials: map[string]any{
				providercore.OpenAIWorkloadCapabilitiesCredentialKey: []any{},
			},
		}

		require.False(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityTextGeneration))
	})

	t.Run("SetupToken 空能力与 OAuth 一样回退为未配置", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeSetupToken,
			Credentials: map[string]any{
				providercore.OpenAIWorkloadCapabilitiesCredentialKey: map[string]any{},
			},
		}

		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityTextGeneration))
	})

	t.Run("非空全 false 能力仍按显式禁用处理", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Credentials: map[string]any{
				providercore.OpenAIWorkloadCapabilitiesCredentialKey: map[string]any{
					string(providercore.OpenAIEndpointCapabilityTextGeneration): false,
				},
			},
		}

		require.False(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityTextGeneration))
	})

	t.Run("类型异常仍按已配置但不含能力处理", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Credentials: map[string]any{
				providercore.OpenAIWorkloadCapabilitiesCredentialKey: "text_generation",
			},
		}

		require.False(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityTextGeneration))
	})

	t.Run("显式数组支持单独关闭文本生成并开启 embeddings", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Credentials: map[string]any{
				"openai_workload_capabilities": []any{"embeddings"},
			},
		}

		require.False(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityTextGeneration))
		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityEmbeddings))
	})

	t.Run("未知能力不应默认放行", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
		}

		require.False(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapability("unknown")))
	})

	t.Run("responses 能力：未探测的 APIKey 默认放行", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
		}

		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityResponses))
	})

	t.Run("responses 能力：历史探测不再排除 APIKey", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Extra:    map[string]any{"openai_responses_probe_status": "unsupported"},
		}

		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityResponses))
		// 非生图路径仍可选中（只要求 chat_completions）。
		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityTextGeneration))
	})

	t.Run("responses 能力：探测确认支持的 APIKey 放行", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Extra:    map[string]any{"openai_responses_probe_status": "supported"},
		}

		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityResponses))
	})

	t.Run("responses 能力：force_chat_completions 覆盖排除 APIKey", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Extra:    map[string]any{"openai_text_route_mode": "force_chat_completions"},
		}

		require.False(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityResponses))
	})

	t.Run("responses 能力：OAuth 提供商不受探测标记影响", func(t *testing.T) {
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Extra:    map[string]any{"openai_responses_probe_status": "unsupported"},
		}

		require.True(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityResponses))
	})

	t.Run("responses 能力：仍需通过 chat_completions 配置集校验", func(t *testing.T) {
		// 未探测（默认支持 responses），但显式能力集未声明 chat_completions。
		provider := &providercore.Record{
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Credentials: map[string]any{
				"openai_workload_capabilities": []any{"embeddings"},
			},
		}

		require.False(t, SupportsOpenAIEndpoint(provider, providercore.OpenAIEndpointCapabilityResponses))
	})
}
