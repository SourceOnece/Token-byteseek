package grok

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveCLIVersionDefaultsToPinnedClientVersion(t *testing.T) {
	t.Setenv(CLIVersionEnv, "")
	// 默认对外版本固定为 CLIClientVersion，CLIStableVersion 仅作为下限。
	require.Equal(t, CLIClientVersion, ResolveCLIVersion())
	require.True(t, IsSupportedCLIVersion(CLIClientVersion))
	require.True(t, IsSupportedCLIVersion(CLIStableVersion))
}

func TestResolveCLIVersionAcceptsValidOverride(t *testing.T) {
	t.Setenv(CLIVersionEnv, "1.0.48-alpha.1")
	require.Equal(t, "1.0.48-alpha.1", ResolveCLIVersion())
}

func TestResolveCLIVersionRejectsUnsafeOrTooOld(t *testing.T) {
	for _, version := range []string{
		"1.0.12",
		"1.0.13-beta.1",
		"1.0.48\r\nX-Injected: true",
		"1.0.013",
		"0.3",
		"1",
	} {
		t.Run(version, func(t *testing.T) {
			t.Setenv(CLIVersionEnv, version)
			require.Equal(t, CLIClientVersion, ResolveCLIVersion())
		})
	}
}

func TestApplyCLIProxyHeaders(t *testing.T) {
	t.Setenv(CLIVersionEnv, "")

	req, err := http.NewRequest(http.MethodPost, "https://cli-chat-proxy.grok.com/v1/responses", nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "legacy-client/1.0")

	ApplyCLIProxyHeaders(req)

	require.Equal(t, CLIClientVersion, req.Header.Get("x-grok-client-version"))
	require.Equal(t, CLIClientIdentifier, req.Header.Get("x-grok-client-identifier"))
	require.Equal(t, CLITokenAuth, req.Header.Get("X-XAI-Token-Auth"))
	require.Equal(t, CLIClientMode, req.Header.Get("x-grok-client-mode"))
	require.Equal(t, "authenticate-response", req.Header.Get("x-authenticateresponse"))
	require.Equal(t, CLIUserAgent(CLIClientVersion), req.Header.Get("User-Agent"))
}

// 共享传输不能保留另一套最低版本或身份头，官方 API 回退必须去掉 CLI 专用标记。
func TestTransportCLIHeadersUseSharedIdentity(t *testing.T) {
	t.Setenv(CLIVersionEnv, CLIStableVersion)
	first, _ := http.NewRequest(http.MethodPost, "https://cli-chat-proxy.grok.com/v1/responses", nil)
	second := first.Clone(first.Context())
	ApplyCLIProxyHeaders(first)
	ApplyTransportCLIHeaders(second)
	require.Equal(t, first.Header, second.Header)
	require.True(t, IsTransportCLIVersionSupported(CLIStableVersion))
	require.False(t, IsTransportCLIVersionSupported("0.2.120"))
}

func TestApplyCLIProxyHeadersLeavesAPIHostUnchanged(t *testing.T) {
	t.Setenv(CLIVersionEnv, "0.2.95")

	req, err := http.NewRequest(http.MethodPost, "https://api.x.ai/v1/responses", nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "direct-api-client/1.0")

	ApplyCLIProxyHeaders(req)

	require.Empty(t, req.Header.Get("x-grok-client-version"))
	require.Empty(t, req.Header.Get("x-grok-client-identifier"))
	require.Empty(t, req.Header.Get("X-XAI-Token-Auth"))
	require.Equal(t, "direct-api-client/1.0", req.Header.Get("User-Agent"))
}
