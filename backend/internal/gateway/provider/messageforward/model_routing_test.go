package messageforward_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/messageforward"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/requeststate"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewayServiceForwardCountTokensAppliesOAuthProviderMappingBeforeNormalization(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", nil)
	c.Request.Header.Set("User-Agent", "third-party-client/1.0")

	body := []byte(`{"model":"group-model","messages":[{"role":"user","content":"hello"}]}`)
	parsed, err := requeststate.ParseGatewayRequest(requeststate.NewRequestBodyRef(body), capability.PlatformAnthropic)
	require.NoError(t, err)

	upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"input_tokens":7}`)),
	}}
	cfg := &messageforward.Options{Configured: true, PreserveContentType: true, ResponseReadLimit: 134217728, MaxLineSize: defaultMaxLineSize}
	svc := newHTTPRuntimeFixture(
		cfg, messageforward.Dependencies{Transport: upstream}, compileResponseHeaderFilter(cfg),
	)
	provider := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, ID: 501,
			Name:        "oauth-count-token-mapping",
			Platform:    capability.PlatformAnthropic,
			Type:        capability.ProviderTypeOAuth,
			Concurrency: 1,
			Credentials: map[string]any{
				"access_token":  "oauth-token",
				"model_mapping": map[string]any{"group-model": "claude-sonnet-4-5"},
			},
			Status:      billing.StatusActive,
			Schedulable: true,
		},
	}

	err = svc.ForwardCountTokens(context.Background(), c, provider, parsed)
	require.NoError(t, err)
	require.Equal(t, "claude-sonnet-4-5-20250929", parsed.Model)
	require.Equal(t, "claude-sonnet-4-5-20250929", gjson.GetBytes(upstream.lastBody, "model").String())
}

func TestGatewayServiceAnthropicCompatibilityForwardersUseFinalOAuthModel(t *testing.T) {
	tests := []struct {
		name string
		path string
		body []byte
		call func(*gatewayhttp.MessagesExecutor, context.Context, *gin.Context, *gatewayprovider.ExecutionProvider, []byte) error
	}{
		{
			name: "chat completions",
			path: "/v1/chat/completions",
			body: []byte(`{"model":"group-model","messages":[{"role":"user","content":"hello"}],"stream":false}`),
			call: func(svc *gatewayhttp.MessagesExecutor, ctx context.Context, c *gin.Context, provider *gatewayprovider.ExecutionProvider, body []byte) error {
				_, err := svc.ForwardAsChatCompletions(ctx, c, provider, body, nil)
				return err
			},
		},
		{
			name: "responses",
			path: "/v1/responses",
			body: []byte(`{"model":"group-model","input":"hello","stream":false}`),
			call: func(svc *gatewayhttp.MessagesExecutor, ctx context.Context, c *gin.Context, provider *gatewayprovider.ExecutionProvider, body []byte) error {
				_, err := svc.ForwardAsResponses(ctx, c, provider, body, nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, tt.path, nil)
			c.Request.Header.Set("User-Agent", "third-party-client/1.0")

			upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"test error"}}`)),
			}}
			cfg := &messageforward.Options{Configured: true, PreserveContentType: true, ResponseReadLimit: 134217728, MaxLineSize: defaultMaxLineSize}
			svc := newHTTPRuntimeFixture(
				cfg, messageforward.Dependencies{Transport: upstream}, compileResponseHeaderFilter(cfg),
			)
			provider := &gatewayprovider.ExecutionProvider{
				Record: providercore.Record{
					LoadLocation: time.LoadLocation, ID: 502,
					Name:        "oauth-compat-mapping",
					Platform:    capability.PlatformAnthropic,
					Type:        capability.ProviderTypeOAuth,
					Concurrency: 1,
					Credentials: map[string]any{
						"access_token":  "oauth-token",
						"model_mapping": map[string]any{"group-model": "claude-sonnet-4-5"},
					},
					Status:      billing.StatusActive,
					Schedulable: true,
				},
			}

			err := tt.call(svc, context.Background(), c, provider, tt.body)
			require.Error(t, err)
			require.Equal(t, "claude-sonnet-4-5-20250929", gjson.GetBytes(upstream.lastBody, "model").String())
		})
	}
}
