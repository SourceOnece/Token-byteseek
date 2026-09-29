package httpapi

import (
	"testing"
	"time"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func TestValidateOpenAIWSBearerTokenAllowsAgentIdentityWithoutStoredToken(t *testing.T) {
	t.Run("Given Agent Identity When a WS path receives no bearer token Then dial-time assertion auth is allowed", func(t *testing.T) {
		provider := &gatewayprovider.ExecutionProvider{
			Record: providercore.Record{
				LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI,
				Type: capability.ProviderTypeOAuth,
				Credentials: map[string]any{
					"auth_mode": providercore.OpenAIAuthModeAgentIdentity,
				},
			},
		}

		require.NoError(t, validateOpenAIWSBearerToken(provider, ""))
	})

	t.Run("Given bearer credentials When a WS path receives no token Then the request is rejected", func(t *testing.T) {
		providers := []*gatewayprovider.ExecutionProvider{
			{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}},
			{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Credentials: map[string]any{"auth_mode": providercore.OpenAIAuthModePersonalAccessToken}}},
			{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey}},
		}

		for _, provider := range providers {
			require.EqualError(t, validateOpenAIWSBearerToken(provider, ""), "token is empty")
		}
	})
}
