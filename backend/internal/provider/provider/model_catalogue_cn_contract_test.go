package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
)

// TestBuildUpstreamModelsRequest_CNProviders 验证“同步上游支持的模型”对国产供应商可用：
// 密钥经 GetOpenAIProtocolAPIKey 读取，/models 端点拼接到提供商 base_url（含默认值）。
func TestBuildUpstreamModelsRequest_CNProviders(t *testing.T) {
	t.Parallel()

	svc := &ModelCatalogue{Options: upstreamModelSyncTestConfig()}
	cases := []struct {
		name     string
		platform string
		mode     string
		wantURL  string
	}{
		{"kimi default", capability.PlatformKimi, "", "https://api.moonshot.cn/v1/models"},
		{"kimi coding", capability.PlatformKimi, providercore.ProviderModeCoding, "https://api.kimi.com/coding/v1/models"},
		{"zhipu default", capability.PlatformZhipu, "", "https://open.bigmodel.cn/api/paas/v4/models"},
		{"zhipu coding", capability.PlatformZhipu, providercore.ProviderModeCoding, "https://open.bigmodel.cn/api/coding/paas/v4/models"},
		{"deepseek", capability.PlatformDeepseek, "", "https://api.deepseek.com/v1/models"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			creds := map[string]any{"api_key": "sk-test"}
			if tc.mode != "" {
				creds["provider_mode"] = tc.mode
			}
			provider := &providercore.Record{ID: 1, Platform: tc.platform, Type: capability.ProviderTypeAPIKey, Credentials: creds}
			req, err := svc.buildUpstreamModelsRequest(context.Background(), provider)
			require.NoError(t, err)
			require.Equal(t, tc.wantURL, req.URL.String())
			require.Equal(t, "Bearer sk-test", req.Header.Get("Authorization"))
		})
	}
}

// TestBuildUpstreamModelsRequest_AnthropicProtocol 模型同步使用协议感知 base。
func TestBuildUpstreamModelsRequest_AnthropicProtocol(t *testing.T) {
	t.Parallel()
	svc := &ModelCatalogue{Options: upstreamModelSyncTestConfig()}
	provider := &providercore.Record{
		ID: 1, Platform: capability.PlatformZhipu, Type: capability.ProviderTypeAPIKey,
		Credentials: map[string]any{
			"api_key":      "sk-test",
			"api_protocol": providercore.APIProtocolAnthropic,
			"base_url":     "https://open.bigmodel.cn/api/anthropic",
		},
	}
	req, err := svc.buildUpstreamModelsRequest(context.Background(), provider)
	require.NoError(t, err)
	require.Equal(t, "https://open.bigmodel.cn/api/paas/v4/models", req.URL.String())
}
