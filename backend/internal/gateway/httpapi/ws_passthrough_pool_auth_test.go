package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	forwardcore "github.com/TokenFlux/TokenRouter/internal/gateway/forward"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestOpenAIGatewayService_APIKeyPassthrough_PoolModeAuthErrorsTriggerFailover 验证池模式认证错误先按提供商配置同号重试。
func TestOpenAIGatewayService_APIKeyPassthrough_PoolModeAuthErrorsTriggerFailover(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		credentials map[string]any
	}{
		{
			name:       "configured_401",
			statusCode: http.StatusUnauthorized,
			credentials: map[string]any{
				"pool_mode_retry_status_codes": []any{float64(http.StatusUnauthorized)},
			},
		},
		{
			name:        "default_403",
			statusCode:  http.StatusForbidden,
			credentials: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(nil))

			upstreamBody := `{"error":{"message":"upstream credential rejected"}}`
			svc := newWSFixture(wsFixtureInputs{options: &wsFixtureOptions{Request: OpenAIRequestOptions{ForceCLI: false}}, health: newUpstreamHealthForTest(transientCooldownProviderRepo{}, &wsFixtureOptions{}, nil, providercore.HealthOptions{}, nil), transport: &auxiliaryHTTPRecorder{resp: &http.Response{
				StatusCode: tt.statusCode,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(upstreamBody)),
			}}})
			credentials := map[string]any{
				"api_key":   "sk-test",
				"base_url":  "https://api.example.test",
				"pool_mode": true,
			}
			for key, value := range tt.credentials {
				credentials[key] = value
			}
			provider := &gatewayprovider.ExecutionProvider{
				Record: providercore.Record{
					LoadLocation: time.LoadLocation, ID: 129, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey, Concurrency: 1,
					Credentials: credentials,
					Extra:       map[string]any{"openai_passthrough": true}, Status: billing.StatusActive, Schedulable: true,
				},
			}

			_, err := svc.Responses.Forward(context.Background(), c, provider, []byte(`{"model":"gpt-5.2","input":"hello"}`))

			var failoverErr *forwardcore.UpstreamFailoverError
			require.ErrorAs(t, err, &failoverErr)
			require.Equal(t, tt.statusCode, failoverErr.StatusCode)
			require.True(t, failoverErr.RetryableOnSameProvider)
			require.False(t, c.Writer.Written(), "池模式认证失败必须在提交响应前进入故障转移")
			require.False(t, IsResponseCommitted(c))
		})
	}
}
