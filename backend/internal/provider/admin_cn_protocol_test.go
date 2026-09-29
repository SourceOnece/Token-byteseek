package provider_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/pkg/apperror"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	protocolcore "github.com/TokenFlux/TokenRouter/internal/protocol"

	"github.com/TokenFlux/TokenRouter/internal/server/httpx"
	"github.com/stretchr/testify/require"
)

// cnProviderProtocolCases 明确列出对外承诺的组合，不从待测校验函数推导期望值。
var cnProviderProtocolCases = []struct {
	platform  string
	mode      string
	protocols []string
}{
	{capability.PlatformDeepseek, providercore.ProviderModePayG, []string{providercore.APIProtocolAdaptive, providercore.APIProtocolChatCompletions, providercore.APIProtocolAnthropic, providercore.APIProtocolResponses}},
	{capability.PlatformKimi, providercore.ProviderModePayG, []string{providercore.APIProtocolAdaptive, providercore.APIProtocolChatCompletions, providercore.APIProtocolAnthropic, providercore.APIProtocolResponses}},
	{capability.PlatformKimi, providercore.ProviderModeCoding, []string{providercore.APIProtocolAdaptive, providercore.APIProtocolChatCompletions, providercore.APIProtocolAnthropic, providercore.APIProtocolResponses}},
	{capability.PlatformZhipu, providercore.ProviderModePayG, []string{providercore.APIProtocolAdaptive, providercore.APIProtocolChatCompletions, providercore.APIProtocolAnthropic}},
	{capability.PlatformZhipu, providercore.ProviderModeCoding, []string{providercore.APIProtocolAdaptive, providercore.APIProtocolChatCompletions, providercore.APIProtocolAnthropic}},
}

// cnProviderTestCredentials 模拟前端提交的自定义端点，验证保存时不会改成官方地址。
func cnProviderTestCredentials(platform, mode, protocol string) map[string]any {
	credentials := map[string]any{
		"api_key":       "sk-test",
		"provider_mode": mode,
		"api_protocol":  protocol,
		"base_url":      "https://relay.example.test/v1",
	}
	if protocol == providercore.APIProtocolAdaptive {
		urls := map[string]any{
			providercore.APIProtocolChatCompletions: "https://relay.example.test/v1",
			providercore.APIProtocolAnthropic:       "https://relay.example.test/anthropic",
		}
		if platform != capability.PlatformZhipu {
			urls[providercore.APIProtocolResponses] = "https://relay.example.test/responses"
		}
		credentials["api_base_urls"] = urls
	}
	return credentials
}

// TestCNProviderProviderProtocolPersistence 覆盖真实创建、编辑及批量凭据更新入口。
func TestCNProviderProviderProtocolPersistence(t *testing.T) {
	t.Parallel()
	for _, tc := range cnProviderProtocolCases {
		for _, protocol := range tc.protocols {
			t.Run(tc.platform+"/"+tc.mode+"/"+protocol, func(t *testing.T) {
				t.Parallel()
				ctx := context.Background()
				repo := &providerServiceTestRepo{}
				svc := newProviderEditorForTest(repo)
				credentials := cnProviderTestCredentials(tc.platform, tc.mode, protocol)
				wantCredentials := cnProviderTestCredentials(tc.platform, tc.mode, protocol)
				delete(wantCredentials, "api_protocol")
				wantProtocols := []protocolcore.ProtocolID{protocolcore.ProtocolOpenAIChatCompletions}
				switch protocol {
				case providercore.APIProtocolAnthropic:
					wantProtocols = []protocolcore.ProtocolID{protocolcore.ProtocolAnthropicMessages}
				case providercore.APIProtocolResponses:
					wantProtocols = []protocolcore.ProtocolID{protocolcore.ProtocolOpenAIResponses}
				case providercore.APIProtocolAdaptive:
					wantProtocols = []protocolcore.ProtocolID{protocolcore.ProtocolAnthropicMessages, protocolcore.ProtocolOpenAIResponses, protocolcore.ProtocolOpenAIChatCompletions}
					if tc.platform == capability.PlatformZhipu {
						wantProtocols = []protocolcore.ProtocolID{protocolcore.ProtocolAnthropicMessages, protocolcore.ProtocolOpenAIChatCompletions}
					}
				}
				wantCredentials[providercore.UpstreamProtocolsKey] = wantProtocols
				if protocol != providercore.APIProtocolAdaptive {
					wantCredentials["api_base_urls"] = map[string]any{protocol: "https://relay.example.test/v1"}
				}

				created, err := svc.CreateProvider(ctx, &providercore.CreateProviderInput{
					Name: "国产平台提供商", Platform: tc.platform, Type: capability.ProviderTypeAPIKey,
					Credentials: credentials,
				})
				require.NoError(t, err)
				require.Equal(t, wantProtocols, created.UpstreamProtocols())
				require.Equal(t, wantCredentials, repo.providers[created.ID].Credentials)

				// 普通字段编辑也会重新校验提供商，必须允许已有自适应提供商正常保存。
				updated, err := svc.UpdateProvider(ctx, created.ID, &providercore.UpdateProviderInput{Name: "已编辑"})
				require.NoError(t, err)
				require.Equal(t, "已编辑", repo.providers[created.ID].Name)
				require.Equal(t, wantCredentials, updated.Credentials)

				// 从传统 Chat 提供商切换到目标协议，覆盖编辑与批量更新的凭据合并。
				repo.providers[created.ID].Credentials = cnProviderTestCredentials(tc.platform, tc.mode, providercore.APIProtocolChatCompletions)
				updated, err = svc.UpdateProvider(ctx, created.ID, &providercore.UpdateProviderInput{Credentials: credentials})
				require.NoError(t, err)
				require.Equal(t, wantCredentials, updated.Credentials)
				repo.providers[created.ID].Credentials = cnProviderTestCredentials(tc.platform, tc.mode, providercore.APIProtocolChatCompletions)
				result, err := svc.BulkUpdateProviders(ctx, &providercore.BulkUpdateProvidersInput{
					ProviderIDs: []int64{created.ID}, Credentials: credentials,
				})
				require.NoError(t, err)
				require.Equal(t, 1, result.Success)
				require.Len(t, repo.bulkUpdates, 1)
				require.Equal(t, wantProtocols, repo.bulkUpdates[0].ProtocolUpdates[created.ID][providercore.UpstreamProtocolsKey])
			})
		}
	}
}

// TestCNProviderBulkProtocolValidationBeforeWrite 混合平台批量修改须在任何写入前拒绝非法组合。
func TestCNProviderBulkProtocolValidationBeforeWrite(t *testing.T) {
	t.Parallel()
	repo := &providerServiceTestRepo{providers: map[int64]*providercore.Record{
		1: {ID: 1, Platform: capability.PlatformKimi, Type: capability.ProviderTypeAPIKey, Credentials: cnProviderTestCredentials(capability.PlatformKimi, providercore.ProviderModePayG, providercore.APIProtocolAdaptive)},
		2: {ID: 2, Platform: capability.PlatformZhipu, Type: capability.ProviderTypeAPIKey, Credentials: cnProviderTestCredentials(capability.PlatformZhipu, providercore.ProviderModePayG, providercore.APIProtocolAdaptive)},
	}}
	svc := newProviderEditorForTest(repo)
	_, err := svc.BulkUpdateProviders(context.Background(), &providercore.BulkUpdateProvidersInput{
		ProviderIDs: []int64{1, 2}, Credentials: map[string]any{"api_protocol": providercore.APIProtocolResponses},
	})
	require.Equal(t, http.StatusBadRequest, httpx.ErrorCode(err))
	require.Equal(t, "CN_PROVIDER_PROTOCOL_INVALID", apperror.Reason(err))
	require.Empty(t, repo.bulkUpdates)
	for _, provider := range repo.providers {
		require.Equal(t, providercore.APIProtocolAdaptive, provider.Credentials["api_protocol"])
	}
}

// TestCNProviderLegacyCredentialDefaults 保留历史读取默认值，普通编辑不补写缺失字段。
func TestCNProviderLegacyCredentialDefaults(t *testing.T) {
	t.Parallel()
	for _, platform := range []string{capability.PlatformDeepseek, capability.PlatformKimi, capability.PlatformZhipu} {
		t.Run(platform, func(t *testing.T) {
			ctx := context.Background()
			repo := &providerServiceTestRepo{providers: map[int64]*providercore.Record{
				1: {ID: 1, Platform: platform, Type: capability.ProviderTypeAPIKey, Credentials: map[string]any{"api_key": "sk-test"}},
			}}
			svc := newProviderEditorForTest(repo)
			updated, err := svc.UpdateProvider(ctx, 1, &providercore.UpdateProviderInput{Name: "历史提供商"})
			require.NoError(t, err)
			require.Equal(t, providercore.ProviderModePayG, updated.GetProviderMode())
			require.Equal(t, []protocolcore.ProtocolID{protocolcore.ProtocolOpenAIChatCompletions}, updated.UpstreamProtocols())
			require.NotContains(t, updated.Credentials, "provider_mode")
			require.NotContains(t, updated.Credentials, "api_protocol")
			created, err := svc.CreateProvider(ctx, &providercore.CreateProviderInput{
				Name: "缺省提供商", Platform: platform, Type: capability.ProviderTypeAPIKey,
				Credentials: map[string]any{"api_key": "sk-test"},
			})
			require.NoError(t, err)
			require.Equal(t, providercore.ProviderModePayG, created.Credentials["provider_mode"])
			require.Equal(t, []protocolcore.ProtocolID{protocolcore.ProtocolOpenAIChatCompletions}, created.UpstreamProtocols())
		})
	}
}
