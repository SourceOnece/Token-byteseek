package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/TokenFlux/TokenRouter/internal/config"
	coderws "github.com/coder/websocket"
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

// 真实 WS 桥接第二轮 429 进入旧安全重放流程：仅开启目标读取连接私有来源。
func TestCodexMetadataRepairHTTPBridgeQuotaSwitch(t *testing.T) {
	for _, nextEnabled := range []bool{false, true} {
		t.Run(fmt.Sprint(nextEnabled), func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			cfg := &config.Config{}
			cfg.Security.URLAllowlist.Enabled = false
			cfg.Security.URLAllowlist.AllowInsecureHTTP = true
			cfg.Gateway.OpenAIWS.Enabled = true
			cfg.Gateway.OpenAIWS.OAuthEnabled = true
			cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
			cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
			cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
			cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

			upstream := &httpUpstreamRecorder{responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type":          []string{"text/event-stream"},
						openAIWSTurnStateHeader: []string{"old-account-state"},
					},
					Body: io.NopCloser(strings.NewReader(
						"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_first\",\"output\":[{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"first-ok\"}]},{\"id\":\"fc_1\",\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"inspect\",\"arguments\":\"{}\"}],\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n",
					)),
				},
				{
					StatusCode: http.StatusTooManyRequests,
					Header:     http.Header{"Retry-After": []string{"60"}},
					Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached"}}`)),
				},
				{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
					Body: io.NopCloser(strings.NewReader(
						"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_second\",\"output\":[{\"id\":\"msg_2\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"second-ok\"}]}],\"usage\":{\"input_tokens\":4,\"output_tokens\":1}}}\n\n",
					)),
				},
			}}
			svc := &OpenAIGatewayService{
				cfg:              cfg,
				httpUpstream:     upstream,
				cache:            &stubGatewayCache{},
				openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
				toolCorrector:    NewCodexToolCorrector(),
			}
			account := &Account{
				ID: 5845, Name: "limited", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
				Status: StatusActive, Schedulable: true, Concurrency: 1,
				Credentials: map[string]any{"chatgpt_account_id": "parent", "access_token": "synthetic"},
				Extra:       map[string]any{"openai_oauth_responses_websockets_v2_mode": OpenAIWSIngressModeHTTPBridge, codexMetadataRepairExtraKey: true},
			}
			nextAccount := *account
			nextAccount.ID = 5846
			nextAccount.Name = "replacement"
			nextAccount.Credentials = map[string]any{"chatgpt_account_id": "replacement", "access_token": "synthetic"}
			nextAccount.Extra = map[string]any{"openai_oauth_responses_websockets_v2_mode": OpenAIWSIngressModeHTTPBridge, codexMetadataRepairExtraKey: nextEnabled}

			serverErrCh := make(chan error, 1)
			retryPayloadCh := make(chan []byte, 1)
			wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := coderws.Accept(w, r, nil)
				if err != nil {
					serverErrCh <- err
					return
				}
				defer func() { _ = conn.CloseNow() }()

				rec := httptest.NewRecorder()
				ginCtx, _ := gin.CreateTestContext(rec)
				ginCtx.Request = r.Clone(r.Context())
				readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
				_, firstMessage, readErr := conn.Read(readCtx)
				cancelRead()
				if readErr != nil {
					serverErrCh <- readErr
					return
				}
				proxyErr := svc.ProxyResponsesWebSocketFromClient(r.Context(), ginCtx, conn, account, "access-token-a", firstMessage, nil)
				var failoverErr *UpstreamFailoverError
				if !errors.As(proxyErr, &failoverErr) {
					serverErrCh <- proxyErr
					return
				}
				retryPayload, retryCurrentTurn := OpenAIWSCurrentTurnRetryPayload(proxyErr)
				if !retryCurrentTurn || len(retryPayload) == 0 {
					serverErrCh <- errors.New("missing current-turn retry payload")
					return
				}
				retryPayloadCh <- retryPayload
				if nextEnabled {
					v, _ := ginCtx.Get(codexMetadataReplayContextKey)
					replay, _ := v.(*codexMetadataReplay)
					if replay == nil || codexMetadataString(replay.stable["thread_id"]) != "t" {
						serverErrCh <- fmt.Errorf("missing replay stable defaults: %#v", replay)
						return
					}
				}
				serverErrCh <- svc.ProxyResponsesWebSocketFromClient(
					r.Context(), ginCtx, conn, &nextAccount, "access-token-b", retryPayload, nil,
				)
			}))
			defer wsServer.Close()

			dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
			clientConn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(wsServer.URL, "http"), nil)
			cancelDial()
			require.NoError(t, err)
			defer func() { _ = clientConn.CloseNow() }()

			writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
			err = clientConn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.6-sol","client_metadata":{"session_id":"s","thread_id":"t","turn_id":"first"},"input":[{"role":"user","content":"first"}]}`))
			cancelWrite()
			require.NoError(t, err)

			readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
			_, completed, err := clientConn.Read(readCtx)
			cancelRead()
			require.NoError(t, err)
			require.Equal(t, "response.completed", gjson.GetBytes(completed, "type").String())

			writeCtx, cancelWrite = context.WithTimeout(context.Background(), 3*time.Second)
			err = clientConn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.6-sol","previous_response_id":"resp_first","client_metadata":{"turn_id":"second"},"input":[{"type":"function_call_output","call_id":"call_1","output":"second"}]}`))
			cancelWrite()
			require.NoError(t, err)

			readCtx, cancelRead = context.WithTimeout(context.Background(), 3*time.Second)
			_, retriedCompleted, err := clientConn.Read(readCtx)
			cancelRead()
			require.NoError(t, err)
			require.Equal(t, "response.completed", gjson.GetBytes(retriedCompleted, "type").String())
			require.Equal(t, "resp_second", gjson.GetBytes(retriedCompleted, "response.id").String())
			_ = clientConn.Close(coderws.StatusNormalClosure, "done")

			select {
			case retryPayload := <-retryPayloadCh:
				require.False(t, gjson.GetBytes(retryPayload, "previous_response_id").Exists())
				require.Equal(t, "gpt-5.6-sol", gjson.GetBytes(retryPayload, "model").String())
				input := gjson.GetBytes(retryPayload, "input")
				require.True(t, input.IsArray())
				require.Len(t, input.Array(), 4)
				require.Contains(t, input.Raw, "first")
				require.Contains(t, input.Raw, "first-ok")
				require.Contains(t, input.Raw, "second")
				require.Equal(t, 1, strings.Count(input.Raw, `"id":"fc_1"`))
				require.Equal(t, 2, strings.Count(input.Raw, `"call_id":"call_1"`))
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for current-turn failover")
			}

			select {
			case proxyErr := <-serverErrCh:
				require.NoError(t, proxyErr)
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for replacement-account completion")
			}
			require.Len(t, upstream.bodies, 3)
			require.Contains(t, string(upstream.bodies[0]), "first")
			require.NotContains(t, string(upstream.bodies[2]), "previous_response_id")
			require.Contains(t, string(upstream.bodies[2]), "second")

			if nextEnabled {
				require.Equal(t, scopeCodexAccountIdentityValue(&nextAccount, 0, "thread", "t"), gjson.GetBytes(upstream.bodies[2], "client_metadata.thread_id").String())
			} else {
				require.False(t, gjson.GetBytes(upstream.bodies[2], "client_metadata.thread_id").Exists(), "关闭账号不接收开启侧默认线程")
			}
		})
	}
}
