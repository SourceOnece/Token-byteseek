package provider_test

import (
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

// 刷新失败的内存桥接仅属于交换凭据；即便通知晚于管理员换凭据，也不能阻断新身份。
func TestRefreshRuntimeBlockIsCredentialScoped(t *testing.T) {
	gateway := provider.NewRuntimeBlockState(time.Now)
	old := &provider.Record{ID: 1, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Credentials: map[string]any{"access_token": "old-fixture"}}
	fresh := *old
	fresh.Credentials = map[string]any{"access_token": "new-fixture"}
	provider.PrepareRefreshFailureNotice(gateway, old)(time.Now().Add(time.Minute), "token_refresh_non_retryable")
	require.True(t, gateway.Blocked(old.ID, func() string { return provider.RefreshCredentialIdentity(old) }))
	require.False(t, gateway.Blocked(fresh.ID, func() string { return provider.RefreshCredentialIdentity(&fresh) }), "旧凭据失败通知阻断了新凭据")
}
