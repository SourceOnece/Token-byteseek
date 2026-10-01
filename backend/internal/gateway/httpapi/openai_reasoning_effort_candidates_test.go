package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// TestOpenAIGatewayServiceForwardOAuthDerivesEffortFromSuffixModel 验证完整后缀型号原样发送且不生成推理档位。
func TestOpenAIGatewayServiceForwardOAuthDerivesEffortFromSuffixModel(t *testing.T) {
	upstream := &auxiliaryHTTPRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"usage":{"input_tokens":1,"output_tokens":2}}`)),
		},
	}
	cfg := &responsesFixtureOptions{}
	cfg.Request.URLPolicy.Enabled = false
	svc := newResponsesFixture(responsesFixtureInputs{options: cfg, transport: upstream})
	provider := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, ID: 11,
			Name:        "openai-oauth-suffix",
			Platform:    capability.PlatformOpenAI,
			Type:        capability.ProviderTypeOAuth,
			Concurrency: 1,
			Credentials: map[string]any{
				"access_token":       "oauth-token",
				"chatgpt_account_id": "chatgpt-acc",
			},
			Status:      billing.StatusActive,
			Schedulable: true,
		},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	body := []byte(`{"model":"gpt-5.3-codex-xhigh","instructions":"suffix-test","input":"hello","stream":false}`)
	result, err := svc.Forward(context.Background(), c, provider, body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "gpt-5.3-codex-xhigh", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Nil(t, result.ReasoningEffort)
}
