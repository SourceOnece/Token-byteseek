//go:build unit

package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 三种客户端入口与三种原生上游协议的九个组合，都必须使用映射后的协议和模型。
func TestOpenCodeNineProtocolCombinations(t *testing.T) {
	for _, ingress := range []string{"responses", "chat/completions", "messages"} {
		for _, protocol := range []string{APIProtocolResponses, APIProtocolChatCompletions, APIProtocolAnthropic} {
			t.Run(ingress+"_"+protocol, func(t *testing.T) {
				account := openCodeMappedTestAccount()
				account.Credentials["model_mapping"] = map[string]any{"public-alias": "glm-5.3"}
				account.Credentials["protocol_rules"] = []any{map[string]any{"pattern": "glm-*", "protocol": protocol}}
				account.Extra = map[string]any{"openai_responses_supported": false}
				for key := range account.Credentials["api_base_urls"].(map[string]any) {
					account.Credentials["api_base_urls"].(map[string]any)[key] = "https://relay.example/native/v1"
				}
				body := []byte(`{"model":"public-alias","stream":false,"input":"test","prompt_cache_key":"matrix-session"}`)
				if ingress != "responses" {
					body = []byte(`{"model":"public-alias","stream":false,"max_tokens":32,"messages":[{"role":"user","content":"test"}],"prompt_cache_key":"matrix-session"}`)
				}
				response := nativeAnthropicBufferedResponse()
				if ingress != "messages" {
					response = nativeAnthropicStreamResponse()
				}
				endpoint := "/native/v1/messages"
				if protocol == APIProtocolChatCompletions {
					endpoint = "/native/v1/chat/completions"
					response = &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"chat_matrix","object":"chat.completion","model":"glm-5.3","choices":[{"index":0,"message":{"role":"assistant","content":"matrix ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}}`))}
				} else if protocol == APIProtocolResponses {
					endpoint = "/native/v1/responses"
					response = &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_matrix\",\"object\":\"response\",\"status\":\"completed\",\"model\":\"glm-5.3\",\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"matrix ok\"}]}],\"usage\":{\"input_tokens\":2,\"output_tokens\":1,\"total_tokens\":3}}}\n\n"))}
				}
				upstream := &httpUpstreamRecorder{resp: response}
				svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
				c := adaptiveProtocolTestContext("/v1/"+ingress, body)
				var result *OpenAIForwardResult
				var err error
				switch ingress {
				case "responses":
					SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)
					result, err = svc.Forward(context.Background(), c, account, body)
				case "messages":
					result, err = svc.ForwardAsAnthropic(context.Background(), c, account, body, "", "")
				default:
					result, err = svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "")
				}
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, "https://relay.example"+endpoint, upstream.lastReq.URL.String())
				require.Equal(t, "glm-5.3", gjson.GetBytes(upstream.lastBody, "model").String())
				require.NotEmpty(t, upstream.lastReq.Header.Get(openCodeSessionHeader))
			})
		}
	}
}

func TestOpenCodeAccountValidationAndProbe(t *testing.T) {
	for _, mode := range []string{AccountModeGo, AccountModeZen} {
		for _, test := range []struct {
			model, protocol, path string
			response              func() *http.Response
		}{
			{"glm-5.3", APIProtocolChatCompletions, "/v1/chat/completions", adaptiveCNChatTestResponse},
			{"gpt-5.6-luna", APIProtocolResponses, "/v1/responses", adaptiveCNResponsesTestResponse},
			{"qwen3.8-max", APIProtocolAnthropic, "/v1/messages", adaptiveCNAnthropicTestResponse},
		} {
			t.Run(fmt.Sprint(mode, "_", test.protocol), func(t *testing.T) {
				account := adaptiveCNAccountTestAccount(802, PlatformOpenCodeGo)
				account.Credentials["account_mode"] = mode
				require.NoError(t, normalizeCNProviderCredentials(account, true))
				svc, upstream := adaptiveCNAccountTestService(account, test.response())
				c, _ := newTestContext()
				require.NoError(t, svc.TestAccountConnection(c, account.ID, test.model, "test", AccountTestModeDefault))
				require.Len(t, upstream.requests, 1)
				require.Equal(t, test.path, upstream.requests[0].URL.Path)
				require.NotEmpty(t, upstream.requests[0].Header.Get(openCodeSessionHeader))
			})
		}
	}
}
