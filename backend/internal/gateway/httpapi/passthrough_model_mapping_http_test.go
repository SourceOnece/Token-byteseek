package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient/tlsfingerprint"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// mappingHTTPTransport 将已构造的真实上游请求交给本机 HTTP 服务，避免外部网络依赖。
type mappingHTTPTransport struct {
	client   *http.Client
	endpoint *url.URL
}

func (p mappingHTTPTransport) Do(request *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	clone := request.Clone(request.Context())
	target := *clone.URL
	target.Scheme = p.endpoint.Scheme
	target.Host = p.endpoint.Host
	clone.URL = &target
	clone.Host = p.endpoint.Host
	return p.client.Do(clone)
}

func (p mappingHTTPTransport) DoWithTLS(request *http.Request, proxy string, id int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return p.Do(request, proxy, id, concurrency)
}

// 验证 HTTP 透传实际发送显式映射后的模型，并保持单跳和请求模型的回填口径。
func TestOpenAIPassthroughHTTPAppliesExplicitModelMappingOnce(t *testing.T) {
	for _, providerType := range []string{capability.ProviderTypeAPIKey, capability.ProviderTypeOAuth} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stream=%v", providerType, stream), func(t *testing.T) {
				observed := make(chan []byte, 1)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					observed <- body
					response := `{"id":"resp_mapping","status":"completed","model":"gpt-5.4","output":[],"usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}`
					w.Header().Set("x-request-id", "mapping-request")
					if gjson.GetBytes(body, "stream").Bool() {
						w.Header().Set("Content-Type", "text/event-stream")
						fmt.Fprintf(w, "data: {\"type\":\"response.completed\",\"response\":%s}\n\ndata: [DONE]\n\n", response)
					} else {
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, response)
					}
				}))
				defer server.Close()
				target, err := url.Parse(server.URL)
				require.NoError(t, err)
				body := []byte(fmt.Sprintf(`{"model":"public-model","stream":%v,"instructions":"test instructions","input":[{"role":"user","content":"hi"}],"custom_extension":{"keep":"exact"}}`, stream))
				original := bytes.Clone(body)
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
				c.Request.Header.Set("Content-Type", "application/json")
				c.Request.Header.Set("User-Agent", "codex_cli_rs/0.1.0")
				service := newResponsesFixture(responsesFixtureInputs{transport: mappingHTTPTransport{client: server.Client(), endpoint: target}, credentials: &provider.OpenAIExecutionCredentials{}})
				value := gatewayadapter.NewExecutionProvider(&provider.Record{ID: 981, Platform: capability.PlatformOpenAI, Type: providerType, Concurrency: 1, Status: provider.StatusActive, Schedulable: true, Extra: map[string]any{"openai_passthrough": true}, Credentials: map[string]any{
					"api_key": "test-key", "base_url": "https://api.openai.com", "access_token": "test-oauth", "chatgpt_account_id": "test-provider",
					"model_mapping":   map[string]any{"public-model": "gpt-5.4", "gpt-5.4": "must-not-map-again"},
					"model_whitelist": []string{"gpt-5.4"},
				}})
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				result, err := service.Forward(ctx, c, value, body)
				require.NoError(t, err)
				require.NotNil(t, result)
				sent := <-observed
				require.Equal(t, "gpt-5.4", gjson.GetBytes(sent, "model").String())
				require.Equal(t, "exact", gjson.GetBytes(sent, "custom_extension.keep").String())
				require.Equal(t, "public-model", result.Model)
				require.Equal(t, "gpt-5.4", result.UpstreamModel)
				require.Contains(t, recorder.Body.String(), `"model":"public-model"`)
				require.Equal(t, original, body)
			})
		}
	}
}
