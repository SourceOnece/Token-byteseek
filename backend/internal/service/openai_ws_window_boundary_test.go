package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 同步假上游让每轮收到请求后才返回，覆盖池模式和透传两种读取方式。
type boundaryCaptureConn struct{ *stagedPassthroughConn }

func (c *boundaryCaptureConn) WriteJSON(ctx context.Context, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.WriteFrame(ctx, coderws.MessageText, payload)
}

func TestContextWindowBoundaryAcrossNativePassthroughAndHTTPBridge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, mode := range []string{"ctx_pool", "passthrough", "http_bridge"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cfg := passthroughLifecycleConfig()
			cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 5
			cfg.Gateway.OpenAIWS.IngressInterTurnIdleTimeoutSeconds = 5
			cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 1
			upstream := &boundaryCaptureConn{newStagedPassthroughConn()}
			defer upstream.Close()
			account := passthroughLifecycleAccount()
			if mode != "passthrough" {
				account.Extra["openai_apikey_responses_websockets_v2_mode"] = "ctx_pool"
			}
			svc := newPassthroughLifecycleService(cfg, upstream.stagedPassthroughConn)
			pool := newOpenAIWSConnPool(cfg)
			defer pool.Close()
			pool.setClientDialerForTest(&stagedPassthroughDialer{conn: upstream})
			svc.openaiWSPool = pool
			wire := func(i int) string {
				return fmt.Sprintf(`{"type":"response.completed","response":{"id":"resp_%d","model":"gpt-5.5","status":"completed","output":[{"type":"function_call","id":"fc_%d","call_id":"call_%d","name":"shell","arguments":"{}"}],"usage":{"input_tokens":1,"output_tokens":1}}}`, i, i, i)
			}
			httpUpstream := &httpUpstreamRecorder{}
			for i := 1; i <= 3; i++ {
				httpUpstream.responses = append(httpUpstream.responses, &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: " + wire(i) + "\n\n"))})
			}
			if mode == "http_bridge" {
				cfg.Gateway.OpenAIWS.HTTPBridgeEnabled = true
				cfg.Gateway.OpenAIWS.HTTPBridgeThresholdBytes = 1
				svc.httpUpstream = httpUpstream
			}
			hookPrev := make(chan string, 3)
			server, done := startPassthroughLifecycleServerWithHooks(t, ctx, svc, account, func(*gin.Context) *OpenAIWSIngressHooks {
				return &OpenAIWSIngressHooks{BeforeRequest: func(_ int, b []byte, _ string, prev string) ([]byte, error) { hookPrev <- prev; return b, nil }}
			})
			defer server.Close()
			client, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
			require.NoError(t, err)
			defer client.CloseNow()
			frames := []string{
				`{"type":"response.create","model":"gpt-5.5","store":false,"client_metadata":{"x-codex-window-id":"a"},"input":[{"role":"user","content":"OLD_WINDOW_TEXT"}]}`,
				`{"type":"response.create","model":"gpt-5.5","store":false,"previous_response_id":"resp_1","client_metadata":{"x-codex-window-id":"b"},"input":[{"role":"user","content":"new context"}]}`,
				`{"type":"response.create","model":"gpt-5.5","store":false,"previous_response_id":"resp_2","client_metadata":{"x-codex-window-id":"b"},"input":[{"type":"function_call_output","call_id":"call_2","output":"ok"}]}`,
			}
			var sent [][]byte
			for i, frame := range frames {
				require.NoError(t, client.Write(ctx, coderws.MessageText, []byte(frame)))
				if mode != "http_bridge" {
					select {
					case b := <-upstream.writes:
						sent = append(sent, b)
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
					upstream.Send(wire(i + 1))
				}
				for {
					_, event, err := client.Read(ctx)
					require.NoError(t, err)
					if gjson.GetBytes(event, "type").String() == "response.completed" {
						break
					}
				}
			}
			_ = client.Close(coderws.StatusNormalClosure, "done")
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if mode == "http_bridge" {
				sent = httpUpstream.bodies
			}
			require.Len(t, sent, 3)
			require.False(t, gjson.GetBytes(sent[1], "previous_response_id").Exists(), "新窗口不得接旧响应链")
			require.NotContains(t, string(sent[1]), "OLD_WINDOW_TEXT")
			require.NotContains(t, string(sent[2]), "OLD_WINDOW_TEXT")
			if mode == "http_bridge" {
				require.Contains(t, string(sent[2]), "call_2", "同一新窗口仍回放工具上下文")
				require.NotContains(t, string(sent[2]), "call_1", "不得回放旧窗口工具")
			} else {
				require.Equal(t, "resp_2", gjson.GetBytes(sent[2], "previous_response_id").String())
			}
			require.Equal(t, "", <-hookPrev, "续聊校验不能再次看到旧窗口的 prev")
		})
	}
}
