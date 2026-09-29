package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	forwardcore "github.com/TokenFlux/TokenRouter/internal/gateway/forward"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type panicOnReadCloser struct{}

func (panicOnReadCloser) Read(_ []byte) (int, error) {
	panic("response body should not be reread")
}

func (panicOnReadCloser) Close() error { return nil }

func TestOpenAIGatewayService_Forward_FailoverReparsesCachedBodyForNextProvider(t *testing.T) {
	tests := []struct {
		name          string
		requestModel  string
		firstMapping  map[string]any
		secondMapping map[string]any
		wantFirst     string
		wantSecond    string
	}{
		{
			name:          "both providers have mapping",
			firstMapping:  map[string]any{"alias-model": "base-model-a"},
			secondMapping: map[string]any{"alias-model": "base-model-b"},
			wantFirst:     "base-model-a",
			wantSecond:    "base-model-b",
		},
		{
			name:         "first provider has mapping second provider has none",
			requestModel: "gpt-5.4-high",
			firstMapping: map[string]any{"gpt-5.4-high": "gpt-5.4"},
			wantFirst:    "gpt-5.4",
			wantSecond:   "gpt-5.4",
		},
		{
			name:          "first provider has no mapping second provider has mapping",
			secondMapping: map[string]any{"alias-model": "base-model-b"},
			wantFirst:     "alias-model",
			wantSecond:    "base-model-b",
		},
		{
			name:          "legacy context cache is ignored when mappings differ",
			firstMapping:  map[string]any{"alias-model": "base-model-a"},
			secondMapping: map[string]any{"alias-model": "base-model-b"},
			wantFirst:     "base-model-a",
			wantSecond:    "base-model-b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestModel := tt.requestModel
			if requestModel == "" {
				requestModel = "alias-model"
			}
			body := []byte(`{"model":"` + requestModel + `","stream":false,"instructions":"cache-test","input":"hello"}`)

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			upstream := &auxiliaryHTTPRecorder{responses: []*http.Response{
				{
					StatusCode: http.StatusTooManyRequests,
					Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-failover-a"}},
					Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","message":"rate limited"}}`)),
				},
				{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-ok-b"}},
					Body:       io.NopCloser(strings.NewReader(`{"id":"resp_123","status":"completed","model":"ok","output":[],"usage":{"input_tokens":1,"output_tokens":1}}`)),
				},
			}}
			svc := newResponsesFixture(responsesFixtureInputs{transport: upstream})

			firstProvider := openAIFailoverCachedBodyTestProvider(1, "provider-a", tt.firstMapping)
			secondProvider := openAIFailoverCachedBodyTestProvider(2, "provider-b", tt.secondMapping)

			_, err := svc.Forward(context.Background(), c, firstProvider, body)
			require.Error(t, err)
			var failoverErr *forwardcore.UpstreamFailoverError
			require.True(t, errors.As(err, &failoverErr))
			require.Len(t, upstream.bodies, 1)
			require.Equal(t, tt.wantFirst, gjson.GetBytes(upstream.bodies[0], "model").String())

			c.Set("openai_parsed_request_body", map[string]any{"model": tt.wantFirst, "stream": true})
			result, err := svc.Forward(context.Background(), c, secondProvider, body)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, upstream.bodies, 2)
			require.Equal(t, tt.wantSecond, gjson.GetBytes(upstream.bodies[1], "model").String())
		})
	}
}

func TestOpenAIGatewayService_HandleFailoverSideEffects_DoesNotRereadResponseBody(t *testing.T) {
	svc := newResponsesFixture(responsesFixtureInputs{})
	provider := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, ID: 88,
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{},
		Body:       panicOnReadCloser{},
	}

	require.NotPanics(t, func() {
		svc.Output.ApplyHTTPFailure(context.Background(), resp, provider, []byte(`{"error":{"type":"rate_limit_error","message":"rate limited"}}`))
	})

	require.False(t, svc.Output.Health.Runtime.Blocked(provider.Record.ID, func() string { return providercore.RefreshCredentialIdentity(provider.View()) }))
	require.True(t, provideradapter.CanRetryOpenAI429(svc.Output.Health.Runtime, provider.View(), nil, nil))
}

func openAIFailoverCachedBodyTestProvider(id int64, name string, mapping map[string]any) *gatewayprovider.ExecutionProvider {
	credentials := map[string]any{"access_token": "oauth-token", "chatgpt_account_id": "chatgpt-provider"}
	if mapping != nil {
		credentials["model_mapping"] = mapping
	}
	return &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, ID: id,
			Name:           name,
			Platform:       capability.PlatformOpenAI,
			Type:           capability.ProviderTypeOAuth,
			Concurrency:    1,
			Credentials:    credentials,
			Status:         billing.StatusActive,
			Schedulable:    true,
			RateMultiplier: new(float64(1)),
		},
	}
}
