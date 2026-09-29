package httpapi

import (
	"context"
	"fmt"
	"github.com/TokenFlux/TokenRouter/internal/gateway/media"
	gp "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// 原生网关入口验证四种图片型号的认证、负载、响应和计费投影。
func TestImagesNativeCodexRoutes(t *testing.T) {
	for _, model := range []string{"gpt-image-1.5", "gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		t.Run(model, func(t *testing.T) {
			body := []byte(fmt.Sprintf(`{"model":%q,"prompt":"原样画图","response_format":"url"}`, model))
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(string(body)))
			c.Request.Header.Set("Content-Type", "application/json")
			parsed, err := media.ParseImageRequest(c.Request.URL.Path, "application/json", body, true)
			require.NoError(t, err)
			transport := &auxiliaryHTTPRecorder{resp: &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"data":[{"b64_json":"aW1hZ2U=","output_format":"png"}],"usage":{"input_tokens":100,"input_tokens_details":{"image_tokens":80,"cached_tokens":50,"cached_tokens_details":{"image_tokens":40,"text_tokens":10}},"output_tokens":200}}`))}}
			executor := newImagesFixture(imagesFixtureInputs{transport: transport})
			account := &gp.ExecutionProvider{Record: provider.Record{LoadLocation: time.LoadLocation, ID: 1, Platform: "openai", Type: "oauth", Credentials: map[string]any{"access_token": "fixture-token", "chatgpt_account_id": "fixture-workspace"}}}
			result, err := executor.ForwardImages(context.Background(), c, account, body, parsed, "")
			require.NoError(t, err)
			require.Equal(t, "/backend-api/codex/images/generations", transport.lastReq.URL.Path)
			require.Equal(t, "Bearer fixture-token", transport.lastReq.Header.Get("Authorization"))
			require.Equal(t, "fixture-workspace", transport.lastReq.Header.Get("chatgpt-account-id"))
			require.Equal(t, model, gjson.GetBytes(transport.lastBody, "model").String())
			require.Equal(t, "原样画图", gjson.GetBytes(transport.lastBody, "prompt").String())
			require.False(t, gjson.GetBytes(transport.lastBody, "tools").Exists())
			require.Equal(t, 1, result.ImageCount)
			require.Equal(t, 200, result.Usage.ImageOutputTokens)
			require.Equal(t, 40, result.Usage.ImageCacheReadTokens)
			require.Equal(t, "data:image/png;base64,aW1hZ2U=", gjson.Get(rec.Body.String(), "data.0.url").String())
		})
	}
}
