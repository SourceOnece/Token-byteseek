package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/sjson"
)

// 扩展原正式测试为缓存键三种关系和流式两态，实际调用所有构造器。
func TestCodexMetadataRepairBusinessPayloadCacheAndStreamMatrix(t *testing.T) {
	for _, cacheCase := range []string{"missing", "same-session", "independent"} {
		for _, stream := range []bool{false, true} {
			for _, route := range []string{"responses", "passthrough", "chat", "messages"} {
				for _, mode := range []string{"off", "device", "session", "full"} {
					t.Run(fmt.Sprintf("%s/%s/%s/stream=%v", route, mode, cacheCase, stream), func(t *testing.T) {
						var originalWire []byte
						var originalHeaders http.Header
						for _, enabled := range []bool{false, true} {
							c, account := metadataRepairTestContext()
							account.Extra[codexMetadataRepairExtraKey] = enabled
							account.Extra[codexFingerprintModeExtraKey] = mode
							account.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
							account.Extra["openai_passthrough"] = route == "passthrough"
							body := map[string]any{
								"model": "gpt-6-astra", "instructions": "Keep user instructions", "stream": stream,
								"input":     []any{map[string]any{"role": "user", "content": "Keep full input"}},
								"reasoning": map[string]any{"effort": "high"}, "max_output_tokens": 1024,
								"include": []string{"reasoning.encrypted_content"}, "store": false,
								"prompt_cache_key": "independent-prompt-cache", "service_tier": "default",
								"tools":           []any{map[string]any{"type": "function", "name": "synthetic_tool", "parameters": map[string]any{"type": "object"}}},
								"client_metadata": map[string]any{"extension": "keep", "session_id": "stable-session", openAIWSTurnMetadataHeader: `{"turn_id":"original-turn"}`},
							}
							if route == "chat" || route == "messages" {
								delete(body, "input")
								body["messages"] = []any{map[string]any{"role": "user", "content": "Keep full input"}}
								body["max_tokens"] = 1024
								if route == "messages" {
									body["tools"] = []any{map[string]any{"name": "synthetic_tool", "input_schema": map[string]any{"type": "object"}}}
								} else {
									body["tools"] = []any{map[string]any{"type": "function", "function": map[string]any{"name": "synthetic_tool", "parameters": map[string]any{"type": "object"}}}}
								}
							}
							if cacheCase == "missing" {
								delete(body, "prompt_cache_key")
							} else if cacheCase == "same-session" {
								body["prompt_cache_key"] = "stable-session"
							}
							raw, err := json.Marshal(body)
							require.NoError(t, err)
							c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(raw))
							c.Request.Header.Set("session-id", "stable-session")
							upstream := &httpUpstreamRecorder{resp: openAICompatSSECompletedResponse("resp_preserve", "gpt-6-astra")}
							svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream, cache: &stubGatewayCache{}}
							switch route {
							case "chat":
								_, err = svc.ForwardAsChatCompletions(context.Background(), c, account, raw, "stable-session", "")
							case "messages":
								_, err = svc.ForwardAsAnthropic(context.Background(), c, account, raw, "stable-session", "")
							default:
								_, err = svc.Forward(context.Background(), c, account, raw)
							}
							require.NoError(t, err)
							require.NotNil(t, upstream.lastReq)
							wire, err := sjson.DeleteBytes(upstream.lastBody, "client_metadata")
							require.NoError(t, err)
							if !enabled {
								originalWire = wire
								originalHeaders = upstream.lastReq.Header.Clone()
								continue
							}
							require.JSONEq(t, string(originalWire), string(wire), "修复不得改变元数据之外的请求业务字段")
							for _, field := range []string{"Authorization", "Chatgpt-Account-Id", "Openai-Beta", "X-Codex-Beta-Features", "X-Codex-Routing-Hint", "User-Agent", "Originator", "Content-Type", "Accept"} {
								require.Equal(t, originalHeaders.Values(field), upstream.lastReq.Header.Values(field), field)
							}
						}
					})
				}
			}
		}
	}
}
