package httpapi_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/pkg/apperror"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	"github.com/TokenFlux/TokenRouter/internal/server/httpx"
	"github.com/stretchr/testify/require"
)

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

// TestCNProviderCredentialValidationRejectsInvalid 保持非法组合的结构化错误边界。
func TestCNProviderCredentialValidationRejectsInvalid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, platform, providerType, mode, protocol, reason string
	}{
		{"智谱原生 Responses", capability.PlatformZhipu, capability.ProviderTypeAPIKey, providercore.ProviderModePayG, providercore.APIProtocolResponses, "CN_PROVIDER_PROTOCOL_INVALID"},
		{"智谱 Coding 原生 Responses", capability.PlatformZhipu, capability.ProviderTypeAPIKey, providercore.ProviderModeCoding, providercore.APIProtocolResponses, "CN_PROVIDER_PROTOCOL_INVALID"},
		{"DeepSeek Coding", capability.PlatformDeepseek, capability.ProviderTypeAPIKey, providercore.ProviderModeCoding, providercore.APIProtocolAdaptive, "CN_PROVIDER_MODE_INVALID"},
		{"非 API Key", capability.PlatformDeepseek, capability.ProviderTypeOAuth, providercore.ProviderModePayG, providercore.APIProtocolAdaptive, "CN_PROVIDER_PROVIDER_TYPE_INVALID"},
		{"未知模式", capability.PlatformKimi, capability.ProviderTypeAPIKey, "unknown", providercore.APIProtocolAdaptive, "CN_PROVIDER_PROVIDER_MODE_INVALID"},
		{"未知协议", capability.PlatformKimi, capability.ProviderTypeAPIKey, providercore.ProviderModePayG, "unknown", "CN_PROVIDER_PROTOCOL_INVALID"},
	}
	for _, tc := range cases {
		for _, create := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/create=%t", tc.name, create), func(t *testing.T) {
				provider := &providercore.Record{
					Platform: tc.platform, Type: tc.providerType,
					Credentials: cnProviderTestCredentials(tc.platform, tc.mode, tc.protocol),
				}
				err := providercore.NormalizeCNProviderCredentials(provider, create)
				require.Error(t, err)
				require.Equal(t, http.StatusBadRequest, httpx.ErrorCode(err))
				require.Equal(t, tc.reason, apperror.Reason(err))
			})
		}
	}
}
