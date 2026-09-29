//go:build unit

package provider

import (
	"context"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func TestAntigravityTokenProvider_GetAccessToken_Upstream(t *testing.T) {
	tokenSource := &AntigravityTokenSource{}

	t.Run("upstream provider with valid api_key", func(t *testing.T) {
		provider := &Record{
			Platform: capability.PlatformAntigravity,
			Type:     capability.ProviderTypeUpstream,
			Credentials: map[string]any{
				"api_key": "sk-test-key-12345",
			},
		}
		token, err := tokenSource.GetAccessToken(context.Background(), provider)
		require.NoError(t, err)
		require.Equal(t, "sk-test-key-12345", token)
	})

	t.Run("upstream provider missing api_key", func(t *testing.T) {
		provider := &Record{
			Platform:    capability.PlatformAntigravity,
			Type:        capability.ProviderTypeUpstream,
			Credentials: map[string]any{},
		}
		token, err := tokenSource.GetAccessToken(context.Background(), provider)
		require.Error(t, err)
		require.Contains(t, err.Error(), "upstream provider missing api_key")
		require.Empty(t, token)
	})

	t.Run("upstream provider with empty api_key", func(t *testing.T) {
		provider := &Record{
			Platform: capability.PlatformAntigravity,
			Type:     capability.ProviderTypeUpstream,
			Credentials: map[string]any{
				"api_key": "",
			},
		}
		token, err := tokenSource.GetAccessToken(context.Background(), provider)
		require.Error(t, err)
		require.Contains(t, err.Error(), "upstream provider missing api_key")
		require.Empty(t, token)
	})

	t.Run("upstream provider with nil credentials", func(t *testing.T) {
		provider := &Record{
			Platform: capability.PlatformAntigravity,
			Type:     capability.ProviderTypeUpstream,
		}
		token, err := tokenSource.GetAccessToken(context.Background(), provider)
		require.Error(t, err)
		require.Contains(t, err.Error(), "upstream provider missing api_key")
		require.Empty(t, token)
	})
}

func TestAntigravityTokenProvider_GetAccessToken_Guards(t *testing.T) {
	tokenSource := &AntigravityTokenSource{}

	t.Run("nil provider", func(t *testing.T) {
		token, err := tokenSource.GetAccessToken(context.Background(), nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "provider is nil")
		require.Empty(t, token)
	})

	t.Run("non-antigravity platform", func(t *testing.T) {
		provider := &Record{
			Platform: capability.PlatformAnthropic,
			Type:     capability.ProviderTypeOAuth,
		}
		token, err := tokenSource.GetAccessToken(context.Background(), provider)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not an antigravity provider")
		require.Empty(t, token)
	})

	t.Run("unsupported provider type", func(t *testing.T) {
		provider := &Record{
			Platform: capability.PlatformAntigravity,
			Type:     capability.ProviderTypeAPIKey,
		}
		token, err := tokenSource.GetAccessToken(context.Background(), provider)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not an antigravity oauth provider")
		require.Empty(t, token)
	})
}
