package openaiattempt

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	forwardcore "github.com/TokenFlux/TokenRouter/internal/gateway/forward"
	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResolveOpenAIUpstreamEndpointPrefersForwardResult(t *testing.T) {
	tests := []struct {
		name            string
		provider        *gatewayprovider.ExecutionProvider
		result          *forwardcore.OpenAIResult
		inboundEndpoint string
		runtimeEndpoint string
		want            string
	}{
		{
			name:            "grok raw chat result overrides stale context",
			provider:        &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformGrok, Type: capability.ProviderTypeOAuth}},
			result:          &forwardcore.OpenAIResult{UpstreamEndpoint: gatewayhttp.EndpointChatCompletions},
			runtimeEndpoint: gatewayhttp.EndpointResponses,
			want:            gatewayhttp.EndpointChatCompletions,
		},
		{
			name:     "grok chat bridged to responses",
			provider: &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformGrok, Type: capability.ProviderTypeOAuth}},
			result:   &forwardcore.OpenAIResult{UpstreamEndpoint: gatewayhttp.EndpointResponses},
			want:     gatewayhttp.EndpointResponses,
		},
		{
			name:     "grok empty result keeps responses default",
			provider: &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformGrok, Type: capability.ProviderTypeOAuth}},
			result:   &forwardcore.OpenAIResult{},
			want:     gatewayhttp.EndpointResponses,
		},
		{
			name:            "grok raw error uses runtime endpoint",
			provider:        &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformGrok, Type: capability.ProviderTypeOAuth}},
			runtimeEndpoint: gatewayhttp.EndpointChatCompletions,
			want:            gatewayhttp.EndpointChatCompletions,
		},
		{
			name:     "openai behavior remains responses",
			provider: &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}},
			result:   &forwardcore.OpenAIResult{},
			want:     gatewayhttp.EndpointResponses,
		},
		{
			name:            "openai api key chat attempt records runtime endpoint",
			provider:        &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey}},
			result:          &forwardcore.OpenAIResult{},
			runtimeEndpoint: gatewayhttp.EndpointChatCompletions,
			want:            gatewayhttp.EndpointChatCompletions,
		},
		{
			name: "openai api key responses attempt records runtime endpoint",
			provider: &gatewayprovider.ExecutionProvider{
				Record: providercore.Record{
					LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI,
					Type:  capability.ProviderTypeAPIKey,
					Extra: map[string]any{"openai_text_route_mode": "force_responses"},
				},
			},
			result:          &forwardcore.OpenAIResult{},
			runtimeEndpoint: gatewayhttp.EndpointResponses,
			want:            gatewayhttp.EndpointResponses,
		},
		{
			name:            "responses fallback records runtime chat endpoint",
			provider:        &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey}},
			result:          &forwardcore.OpenAIResult{},
			inboundEndpoint: gatewayhttp.EndpointResponses,
			runtimeEndpoint: gatewayhttp.EndpointChatCompletions,
			want:            gatewayhttp.EndpointChatCompletions,
		},
		{
			name:            "messages native path records runtime responses endpoint",
			provider:        &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey}},
			result:          &forwardcore.OpenAIResult{},
			inboundEndpoint: gatewayhttp.EndpointMessages,
			runtimeEndpoint: gatewayhttp.EndpointResponses,
			want:            gatewayhttp.EndpointResponses,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			inboundEndpoint := tt.inboundEndpoint
			if inboundEndpoint == "" {
				inboundEndpoint = gatewayhttp.EndpointChatCompletions
			}
			c.Request = httptest.NewRequest(http.MethodPost, inboundEndpoint, nil)
			c.Set("_gateway_inbound_endpoint", inboundEndpoint)
			gatewayhttp.SetActualOpenAIUpstreamEndpoint(c, tt.runtimeEndpoint)
			require.Equal(t, tt.want, ResolveOpenAIUpstreamEndpoint(c, tt.provider, tt.result))
		})
	}
}
