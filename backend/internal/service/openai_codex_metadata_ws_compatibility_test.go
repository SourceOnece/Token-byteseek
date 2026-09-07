package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 三种 WS 模式、四种指纹、开关两态的预热与续接；只使用本地服务和假上游。
func TestCodexMetadataRepairWSPrewarmCompatibility(t *testing.T) {
	for _, mode := range []string{OpenAIWSIngressModeCtxPool, OpenAIWSIngressModePassthrough, OpenAIWSIngressModeHTTPBridge} {
		for _, fingerprint := range []string{"off", "device", "session", "full"} {
			for _, enabled := range []bool{false, true} {
				t.Run(fmt.Sprintf("mode=%s/fingerprint=%s/repair=%v", mode, fingerprint, enabled), func(t *testing.T) {
					_, account := metadataRepairTestContext()
					account.Extra[codexMetadataRepairExtraKey] = enabled
					account.Extra[codexFingerprintModeExtraKey] = fingerprint
					account.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
					account.Extra["openai_oauth_responses_websockets_v2_mode"] = mode
					// 先投影同租户父请求，验证随机收敛下也不以原始父编号冒充出站编号。
					parentC, _ := metadataRepairTestContext()
					parentThread, parentTurn := "parent-"+t.Name(), "parent-turn-"+t.Name()
					parentBody, _ := json.Marshal(map[string]any{"client_metadata": map[string]any{"session_id": parentThread, "thread_id": parentThread, "turn_id": parentTurn}})
					parent := buildCodexMetadataRepair(parentC, account, parentBody, "", true)
					if parent != nil {
						_, parentErr := parent.applyRaw(parentBody)
						require.NoError(t, parentErr)
					}
					cfg := newOpenAIWSV2TestConfig()
					cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
					cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 1
					cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
					cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 1
					cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
					conn := &openAIWSCaptureConn{events: [][]byte{
						[]byte(`{"type":"response.completed","response":{"id":"resp_audit_prewarm","model":"gpt-6-astra","usage":{"input_tokens":1,"output_tokens":1}}}`),
						[]byte(`{"type":"response.completed","response":{"id":"resp_audit_turn","model":"gpt-6-astra","usage":{"input_tokens":1,"output_tokens":1}}}`),
					}}
					dialer := &openAIWSCaptureDialer{conn: conn}
					pool := newOpenAIWSConnPool(cfg)
					defer pool.Close()
					pool.setClientDialerForTest(dialer)
					upstream := &httpUpstreamRecorder{responses: []*http.Response{openAICompatSSECompletedResponse("resp_audit_prewarm", "gpt-6-astra"), openAICompatSSECompletedResponse("resp_audit_turn", "gpt-6-astra")}}
					svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream, cache: &stubGatewayCache{}, openaiWSResolver: NewOpenAIWSProtocolResolver(cfg), toolCorrector: NewCodexToolCorrector(), openaiWSPool: pool, openaiWSPassthroughDialer: dialer}
					if mode == OpenAIWSIngressModePassthrough {
						svc.openaiWSPassthroughDialer = &metadataRepairTurnDialer{conn: &metadataRepairTurnConn{openAIWSCaptureConn: conn, ready: make(chan struct{}, 2)}}
					}
					errCh := make(chan error, 1)
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						client, err := coderws.Accept(w, r, nil)
						if err != nil {
							errCh <- err
							return
						}
						defer client.CloseNow()
						c, _ := gin.CreateTestContext(httptest.NewRecorder())
						c.Request = r.Clone(r.Context())
						_, first, err := client.Read(r.Context())
						if err != nil {
							errCh <- err
							return
						}
						errCh <- svc.ProxyResponsesWebSocketFromClient(r.Context(), c, client, account, "synthetic-token", first, nil)
					}))
					defer server.Close()
					ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
					defer cancel()
					headers := make(http.Header)
					headers.Set("session_id", "stable-session")
					headers.Set("x-client-request-id", "stable-thread")
					headers.Set("x-codex-window-id", "stable-window")
					headers.Set("User-Agent", "codex-tui/0.144.1")
					headers.Set(codexParentThreadIDHeader, parentThread)
					headers.Set(openAISubagentHeader, "collab_spawn")
					headers.Set(openAIWSTurnMetadataHeader, `{"request_kind":"prewarm","installation_id":"stable-installation","session_id":"stable-session","thread_id":"stable-thread"}`)
					client, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), &coderws.DialOptions{HTTPHeader: headers})
					require.NoError(t, err)
					defer client.CloseNow()
					for index, kind := range []string{"prewarm", "turn"} {
						metadata, _ := json.Marshal(map[string]any{"request_kind": kind, "installation_id": "stable-installation", "session_id": "stable-session", "thread_id": "stable-thread", "turn_id": fmt.Sprintf("turn-%d", index), "parent_thread_id": parentThread, "parent_turn_id": parentTurn, "subagent_kind": "thread_spawn", "tool_namespaces_info": map[string]any{"tools": []any{}}})
						body := map[string]any{"type": "response.create", "model": "gpt-6-astra", "instructions": "Synthetic test", "store": false, "input": []any{map[string]any{"role": "user", "content": "test"}}, "client_metadata": map[string]any{"installation_id": "stable-installation", "session_id": "stable-session", "thread_id": "stable-thread", openAIWSTurnMetadataHeader: string(metadata)}}
						if index == 0 {
							body["generate"] = false
						} else {
							body["previous_response_id"] = "resp_audit_prewarm"
						}
						raw, _ := json.Marshal(body)
						require.NoError(t, client.Write(ctx, coderws.MessageText, raw))
						for {
							_, payload, readErr := client.Read(ctx)
							require.NoError(t, readErr, "正常回合必须收到完成事件")
							if gjson.GetBytes(payload, "type").String() == "response.completed" {
								break
							}
						}
					}
					_ = client.CloseNow()
					select {
					case <-errCh:
					case <-ctx.Done():
						t.Fatal("本机测试未退出")
					}
					if mode == OpenAIWSIngressModeCtxPool {
						require.Equal(t, 1, dialer.DialCount(), "预热和正常回合复用同一上游连接")
					}
					if enabled {
						var finalBody []byte
						if mode == OpenAIWSIngressModeHTTPBridge {
							finalBody = upstream.lastBody
							require.Equal(t, parent.metadata["thread_id"], upstream.lastReq.Header.Get(codexParentThreadIDHeader))
						} else {
							finalBody = []byte(requestToJSONString(conn.lastWrite))
						}
						value := gjson.Parse(gjson.GetBytes(finalBody, "client_metadata.x-codex-turn-metadata").String())
						require.Equal(t, parent.metadata["thread_id"], value.Get("parent_thread_id").String())
						require.Equal(t, parent.metadata["turn_id"], value.Get("parent_turn_id").String())
						require.Equal(t, "thread_spawn", value.Get("subagent_kind").String())
					}
				})
			}
		}
	}
}
