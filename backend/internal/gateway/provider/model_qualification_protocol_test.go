package provider

import (
	"net/http"
	"testing"

	protocolcore "github.com/TokenFlux/TokenRouter/internal/protocol"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	"github.com/TokenFlux/TokenRouter/internal/server/httpx"
	"github.com/stretchr/testify/require"
)

// 目录、保存与实际路线共享同一矩阵，显式空集合和平台拒绝均不能被默认值覆盖。
func TestProtocolNativeMatrixAndSave(t *testing.T) {
	for _, tc := range []struct {
		platform, kind, auth string
		count                int
	}{
		{capability.PlatformAnthropic, capability.ProviderTypeAPIKey, "", 1},
		{capability.PlatformAnthropic, capability.ProviderTypeBedrock, "", 1},
		{capability.PlatformOpenAI, capability.ProviderTypeAPIKey, "", 8},
		{capability.PlatformOpenAI, capability.ProviderTypeOAuth, "", 5},
		{capability.PlatformOpenAI, capability.ProviderTypeOAuth, providercore.OpenAIAuthModePersonalAccessToken, 3},
		{capability.PlatformOpenAI, capability.ProviderTypeOAuth, providercore.OpenAIAuthModeAgentIdentity, 4},
		{capability.PlatformDeepseek, capability.ProviderTypeAPIKey, "", 3},
		{capability.PlatformKimi, capability.ProviderTypeAPIKey, "", 3},
		{capability.PlatformZhipu, capability.ProviderTypeAPIKey, "", 2},
		{capability.PlatformGemini, capability.ProviderTypeAPIKey, "", 2},
		{capability.PlatformGemini, capability.ProviderTypeServiceAccount, "", 2},
		{capability.PlatformGemini, capability.ProviderTypeOAuth, "", 1},
		{capability.PlatformAntigravity, capability.ProviderTypeOAuth, "", 1},
		{capability.PlatformAntigravity, capability.ProviderTypeAPIKey, "", 0},
		{capability.PlatformGrok, capability.ProviderTypeAPIKey, "", 11},
		{capability.PlatformGrok, capability.ProviderTypeOAuth, "", 11},
		{capability.PlatformQoder, capability.ProviderTypeCosy, "", 1},
	} {
		t.Run(tc.platform+"/"+tc.kind+"/"+tc.auth, func(t *testing.T) {
			provider := &providercore.Record{Platform: tc.platform, Type: tc.kind, Credentials: map[string]any{"auth_mode": tc.auth}}
			options := provider.NativeProtocolOptions()
			require.Len(t, options, tc.count)
			for _, protocol := range options {
				provider.Credentials[providercore.UpstreamProtocolsKey] = []string{string(protocol)}
				require.NoError(t, providercore.NormalizeProviderProtocols(provider))
				require.Equal(t, []protocolcore.ProtocolID{protocol}, provider.UpstreamProtocols())
				target, ok := (ModelPolicy{Record: provider}).ProtocolRoute(nil, protocol)
				require.True(t, ok)
				require.Equal(t, protocol, target)
			}
			provider.Credentials[providercore.UpstreamProtocolsKey] = []string{}
			require.NoError(t, providercore.NormalizeProviderProtocols(provider))
			require.Empty(t, provider.UpstreamProtocols())
			provider.Credentials[providercore.UpstreamProtocolsKey] = []string{"unknown"}
			require.Equal(t, http.StatusBadRequest, httpx.ErrorCode(providercore.NormalizeProviderProtocols(provider)))
		})
	}
}
