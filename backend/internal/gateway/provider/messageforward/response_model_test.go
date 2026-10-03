package messageforward_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/gateway/forward"
	gatewaycapture "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/messageforward"
	"github.com/TokenFlux/TokenRouter/internal/protocol/openai"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// TestAnthropicConversionPreservesResponseModel 验证四条转换路径区分客户端别名、出站模型和响应声明。
func TestAnthropicConversionPreservesResponseModel(t *testing.T) {
	const clientModel = "client-alias"
	const sentModel = "claude-fable-5-1"
	for _, wire := range []string{"responses", "chat"} {
		for _, stream := range []bool{false, true} {
			for _, declaration := range []struct {
				name  string
				value any
				want  string
			}{
				{name: "different", value: " claude-runtime-version ", want: "claude-runtime-version"},
				{name: "same", value: sentModel, want: sentModel},
				{name: "missing"},
			} {
				t.Run(fmt.Sprintf("%s/stream=%t/%s", wire, stream, declaration.name), func(t *testing.T) {
					message := map[string]any{
						"id": "msg_1", "type": "message", "role": "assistant",
						"content": []any{}, "usage": map[string]any{"input_tokens": 100},
					}
					if declaration.value != nil {
						message["model"] = declaration.value
					}
					start, err := json.Marshal(map[string]any{"type": "message_start", "message": message})
					require.NoError(t, err)
					upstreamBody := "event: message_start\ndata: " + string(start) + "\n\n" +
						"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":1}}\n\n" +
						"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
					transport := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": {"text/event-stream"}},
						Body:       io.NopCloser(strings.NewReader(upstreamBody)),
					}}
					executor := newHTTPRuntimeFixture(&messageforward.Options{
						Configured: true, PreserveContentType: true, ResponseReadLimit: 134217728,
					}, messageforward.Dependencies{Transport: transport}, nil)
					target := &gatewaycapture.ExecutionProvider{Record: provider.Record{
						LoadLocation: time.LoadLocation, ID: 1, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeAPIKey,
						Credentials: map[string]any{
							"api_key": "test-key", "base_url": "https://api.anthropic.com",
							"model_mapping": map[string]any{clientModel: sentModel},
						},
					}}
					recorder := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(recorder)
					c.Request = httptest.NewRequest(http.MethodPost, "/v1/"+wire, nil)
					request := map[string]any{"model": clientModel, "stream": stream}
					if wire == "responses" {
						request["input"] = "hello"
					} else {
						request["messages"] = []any{map[string]any{"role": "user", "content": "hello"}}
					}
					body, err := json.Marshal(request)
					require.NoError(t, err)
					var result *forward.MessagesResult
					if wire == "responses" {
						result, err = executor.ForwardAsResponses(c.Request.Context(), c, target, body, nil)
					} else {
						result, err = executor.ForwardAsChatCompletions(c.Request.Context(), c, target, body, nil)
					}
					require.NoError(t, err)
					require.NotNil(t, result)
					require.Equal(t, clientModel, result.Model)
					require.Equal(t, sentModel, result.UpstreamModel)
					require.Equal(t, sentModel, gjson.GetBytes(transport.lastBody, "model").String())
					require.Equal(t, declaration.want, result.UpstreamResponseModel)
					require.Equal(t, 100, result.Usage.InputTokens)
					require.Equal(t, 1, result.Usage.OutputTokens)
					captured := gatewaycapture.ProjectMessagesCompletionResult(result, &target.Record)
					require.Equal(t, declaration.want, captured.UpstreamResponseModel)

					// 客户端继续看到自己的别名，原始上游声明仅随完成结果保存。
					if !stream {
						require.Equal(t, clientModel, gjson.Get(recorder.Body.String(), "model").String())
					} else {
						models := 0
						openai.ForEachOpenAISSEFrame(recorder.Body.String(), func(_ string, payload []byte) {
							for _, path := range []string{"model", "response.model"} {
								if model := gjson.GetBytes(payload, path); model.Exists() {
									models++
									require.Equal(t, clientModel, model.String())
								}
							}
						})
						require.Positive(t, models)
					}
				})
			}
		}
	}
}
