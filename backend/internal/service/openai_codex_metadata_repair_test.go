package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/pkg/tlsfingerprint"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 全部使用合成凭据及内存上游，不发真实请求或触碰生产账号。
func metadataRepairTestContext() (*gin.Context, *Account) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	a := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"access_token": "synthetic-token", "chatgpt_account_id": "synthetic-account"},
		Extra:       map[string]any{codexMetadataRepairExtraKey: true}}
	return c, a
}

func TestCodexMetadataRepairOptIn(t *testing.T) {
	for _, value := range []any{nil, false, "true", 1} {
		_, a := metadataRepairTestContext()
		a.Extra[codexMetadataRepairExtraKey] = value
		require.False(t, a.IsCodexMetadataRepairEnabled())
	}
	_, a := metadataRepairTestContext()
	require.True(t, a.IsCodexMetadataRepairEnabled())
	a.Type = AccountTypeSetupToken
	require.True(t, a.IsCodexMetadataRepairEnabled())
	a.Type = AccountTypeAPIKey
	require.False(t, a.IsCodexMetadataRepairEnabled())
	a.Type, a.Platform = AccountTypeOAuth, PlatformAnthropic
	require.False(t, a.IsCodexMetadataRepairEnabled())
	var absent *Account
	require.False(t, absent.IsCodexMetadataRepairEnabled())
}

func TestCodexMetadataRepairSourcesAndCurrentTurn(t *testing.T) {
	c, a := metadataRepairTestContext()
	c.Request.Header.Set(openAIWSTurnMetadataHeader, `{"turn_id":"old-turn","session_id":"session-a","header_field":1,"sandbox":"old"}`)
	body := []byte(`{"model":"gpt-5.4","client_metadata":{"extension":"preserve","x-codex-turn-metadata":"{\"turn_id\":\"new-turn\",\"sandbox\":\"actual\",\"body_field\":2}"},"input":["unchanged"]}`)
	snapshot := buildCodexMetadataRepair(c, a, body, "", true)
	require.NotNil(t, snapshot)
	got, err := snapshot.applyRaw(body)
	require.NoError(t, err)
	require.Equal(t, "preserve", gjson.GetBytes(got, "client_metadata.extension").String())
	turn := gjson.Parse(snapshot.headers.Get(openAIWSTurnMetadataHeader))
	require.Equal(t, "actual", turn.Get("sandbox").String())
	require.Equal(t, int64(1), turn.Get("header_field").Int())
	require.Equal(t, int64(2), turn.Get("body_field").Int())
	require.Equal(t, scopeCodexAccountIdentityValue(a, 0, "turn", "new-turn"), turn.Get("turn_id").String())
	require.Equal(t, turn.Raw, gjson.GetBytes(got, "client_metadata.x-codex-turn-metadata").String())
	require.Equal(t, `["unchanged"]`, gjson.GetBytes(got, "input").Raw)
	next := buildCodexMetadataRepair(c, a, []byte(`{"client_metadata":{"x-codex-turn-metadata":"{\"turn_id\":\"next-turn\"}"}}`), "", false)
	require.NotNil(t, next)
	nextTurn := gjson.Parse(next.headers.Get(openAIWSTurnMetadataHeader))
	require.False(t, nextTurn.Get("header_field").Exists())
	require.False(t, nextTurn.Get("sandbox").Exists())
	require.NotEqual(t, turn.Get("turn_id").String(), nextTurn.Get("turn_id").String())
	require.Equal(t, turn.Get("session_id").String(), nextTurn.Get("session_id").String())
}

func TestCodexMetadataRepairNoFabricatedEnvironmentAndRetry(t *testing.T) {
	c, a := metadataRepairTestContext()
	body := []byte(`{"model":"gpt-5.4","input":[]}`)
	prepareCodexMetadataRepair(c, a, body, "session-hint")
	snapshot := stagedCodexMetadataRepair(c, a)
	require.NotNil(t, snapshot)
	first, err := snapshot.applyRaw(body)
	require.NoError(t, err)
	second, err := snapshot.applyRaw(body)
	require.NoError(t, err)
	require.Equal(t, string(first), string(second))
	prepareCodexMetadataRepair(c, a, body, "session-hint")
	require.Same(t, snapshot, stagedCodexMetadataRepair(c, a), "handler 重新进入 Forward 仍复用同请求快照")
	turn := gjson.Parse(snapshot.headers.Get(openAIWSTurnMetadataHeader))
	require.NotEmpty(t, turn.Get("turn_id").String())
	require.True(t, turn.Get("turn_started_at_unix_ms").Int() > 0)
	for _, field := range []string{"installation_id", "thread_id", "window_id", "sandbox", "thread_source"} {
		require.False(t, turn.Get(field).Exists(), field)
	}
	m := map[string]any{"input": "unchanged"}
	snapshot.applyMap(m)
	m["client_metadata"].(map[string]any)["turn_id"] = "mutated"
	third, err := snapshot.applyRaw(body)
	require.NoError(t, err)
	require.Equal(t, string(first), string(third))
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	prepareCodexMetadataRepair(c, a, body, "session-hint")
	require.NotEqual(t, snapshot.headers.Get(openAIWSTurnMetadataHeader), stagedCodexMetadataRepair(c, a).headers.Get(openAIWSTurnMetadataHeader), "新请求生成新回合")
}

func TestCodexMetadataRepairDeviceFingerprintAndIsolation(t *testing.T) {
	for _, mode := range []string{"off", "device", "session", "full"} {
		t.Run(mode, func(t *testing.T) {
			c, a := metadataRepairTestContext()
			a.Extra["openai_device_id"] = "real-configured-device"
			a.Extra[codexFingerprintModeExtraKey] = mode
			a.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
			c.Request.Header.Set("session-id", "session-a")
			snapshot := buildCodexMetadataRepair(c, a, []byte(`{"input":[]}`), "", true)
			require.NotNil(t, snapshot)
			if mode == "session" {
				require.Equal(t, scopeCodexAccountIdentityValue(a, 0, "thread", "session-a"), snapshot.metadata["thread_id"], "无 thread 时使用原始会话来源并保持租户隔离")
			}
			turn := gjson.Parse(snapshot.headers.Get(openAIWSTurnMetadataHeader))
			for _, identity := range codexMetadataRepairIdentity {
				if v := turn.Get(identity.field).String(); v != "" {
					require.Equal(t, v, snapshot.headers.Get(identity.aliases[0]))
					require.Equal(t, v, snapshot.metadata[identity.aliases[0]])
				}
			}
		})
	}
	c, a := metadataRepairTestContext()
	prepareCodexMetadataRepair(c, a, []byte(`{}`), "")
	require.NotNil(t, stagedCodexMetadataRepair(c, a))
	other := *a
	other.ID++
	require.Nil(t, stagedCodexMetadataRepair(c, &other))
	other.Extra = map[string]any{}
	prepareCodexMetadataRepair(c, &other, []byte(`{}`), "")
	require.Nil(t, stagedCodexMetadataRepair(c, a))
}

func TestCodexMetadataRepairCompactAndInvalidUntouched(t *testing.T) {
	c, a := metadataRepairTestContext()
	for _, body := range []string{`{"client_metadata":[]}`, `{"client_metadata":"bad"}`, `{"client_metadata":{"x-codex-turn-metadata":"bad"}}`, `{"client_metadata":{"large":"` + strings.Repeat("x", codexMetadataRepairBodyMaxBytes) + `"}}`} {
		require.Nil(t, buildCodexMetadataRepair(c, a, []byte(body), "", true))
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	prepareCodexMetadataRepair(c, a, []byte(`{}`), "")
	require.Nil(t, stagedCodexMetadataRepair(c, a))
}

// 官方完整库存只放 body；非 ASCII、大整数和用户扩展不得因 header 投影而丢失。
func TestCodexMetadataRepairOfficialProjection(t *testing.T) {
	c, a := metadataRepairTestContext()
	turn := map[string]any{"turn_id": "current", "agent_name": "鹈鹕🚲", "large_ordinal": json.Number("9007199254740993"), "tool_namespaces_info": map[string]any{"tools": strings.Repeat("tool", 6000)}}
	raw, err := json.Marshal(turn)
	require.NoError(t, err)
	body, err := json.Marshal(map[string]any{"model": "gpt-6-astra", "client_metadata": map[string]any{openAIWSTurnMetadataHeader: string(raw)}})
	require.NoError(t, err)
	snapshot := buildCodexMetadataRepair(c, a, body, "", true)
	require.NotNil(t, snapshot)
	full := gjson.Parse(snapshot.metadata[openAIWSTurnMetadataHeader].(string))
	header := snapshot.headers.Get(openAIWSTurnMetadataHeader)
	require.True(t, full.Get("tool_namespaces_info").Exists())
	require.False(t, gjson.Get(header, "tool_namespaces_info").Exists())
	require.Equal(t, "鹈鹕🚲", full.Get("agent_name").String())
	require.Equal(t, "9007199254740993", full.Get("large_ordinal").Raw)
	require.Equal(t, "turn", full.Get("request_kind").String())
	require.Equal(t, full.Get("turn_id").String(), gjson.Get(header, "turn_id").String())
	for _, r := range header {
		require.Less(t, r, rune(128))
	}
	for _, payload := range []map[string]any{{"input": []any{map[string]any{"type": "compaction_trigger"}}}, {"generate": false}} {
		encoded, _ := json.Marshal(payload)
		next := buildCodexMetadataRepair(c, a, encoded, "", true)
		require.NotNil(t, next)
		want := "compaction"
		if payload["generate"] == false {
			want = "prewarm"
		}
		require.Equal(t, want, gjson.Get(next.metadata[openAIWSTurnMetadataHeader].(string), "request_kind").String())
	}
}

func TestCodexMetadataRepairOversizedCompatibilityHeader(t *testing.T) {
	c, a := metadataRepairTestContext()
	raw, _ := json.Marshal(map[string]any{"turn_id": "turn-a", "extension": strings.Repeat("x", codexMetadataRepairMaxBytes+50), "request_kind": "memory"})
	body, _ := json.Marshal(map[string]any{"client_metadata": map[string]any{openAIWSTurnMetadataHeader: string(raw)}})
	snapshot := buildCodexMetadataRepair(c, a, body, "", true)
	require.NotNil(t, snapshot)
	headers := http.Header{http.CanonicalHeaderKey(openAIWSTurnMetadataHeader): []string{"old"}}
	snapshot.applyHeaders(headers)
	require.Empty(t, headers.Get(openAIWSTurnMetadataHeader))
	full := snapshot.metadata[openAIWSTurnMetadataHeader].(string)
	require.Equal(t, "memory", gjson.Get(full, "request_kind").String())
	require.Equal(t, codexMetadataRepairMaxBytes+50, len(gjson.Get(full, "extension").String()))
	a.Extra[codexMetadataRepairExtraKey] = false
	require.Nil(t, buildCodexMetadataRepair(c, a, body, "", true))
}

func TestCodexMetadataRepairPrewarmDoesNotMutateTurn(t *testing.T) {
	c, a := metadataRepairTestContext()
	snapshot := buildCodexMetadataRepair(c, a, []byte(`{"model":"gpt-6-astra"}`), "", true)
	require.NotNil(t, snapshot)
	original := map[string]any{}
	snapshot.applyMap(original)
	prewarm := map[string]any{"client_metadata": original["client_metadata"], "generate": false}
	applyCodexMetadataPrewarmKind(a, prewarm)
	require.Equal(t, "prewarm", gjson.Get(prewarm["client_metadata"].(map[string]any)[openAIWSTurnMetadataHeader].(string), "request_kind").String())
	require.Equal(t, "turn", gjson.Get(original["client_metadata"].(map[string]any)[openAIWSTurnMetadataHeader].(string), "request_kind").String())
	a.Extra[codexMetadataRepairExtraKey] = false
	applyCodexMetadataPrewarmKind(a, original)
	require.Equal(t, "turn", gjson.Get(original["client_metadata"].(map[string]any)[openAIWSTurnMetadataHeader].(string), "request_kind").String())
}

func TestCodexMetadataRepairHeaderSafetyAndPoolCompatibility(t *testing.T) {
	c, a := metadataRepairTestContext()
	for _, metadata := range []string{`{"turn_id":"bad\r\nvalue"}`, `{"installation_id":"bad\u0000value"}`} {
		body, err := json.Marshal(map[string]any{"client_metadata": map[string]any{openAIWSTurnMetadataHeader: metadata}})
		require.NoError(t, err)
		require.Nil(t, buildCodexMetadataRepair(c, a, body, "", true))
	}
	c.Request.Header.Set(openAIWSTurnMetadataHeader, "not-json")
	require.NotNil(t, buildCodexMetadataRepair(c, a, []byte(`{}`), "", true), "坏兼容头不阻断合法请求的修复")
	c.Request.Header.Del(openAIWSTurnMetadataHeader)
	snapshot := buildCodexMetadataRepair(c, a, []byte(`{}`), "session-one", true)
	headers := http.Header{"x-codex-turn-metadata": []string{"stale"}, "X-Codex-Turn-Metadata": []string{"other"}}
	snapshot.applyHeaders(headers)
	count := 0
	for key := range headers {
		if strings.EqualFold(key, openAIWSTurnMetadataHeader) {
			count++
		}
	}
	require.Equal(t, 1, count)
	enabledKey := normalizeOpenAIWSHandshakeCompatibility(a, headers)
	a.Extra[codexMetadataRepairExtraKey] = false
	require.NotEqual(t, enabledKey, normalizeOpenAIWSHandshakeCompatibility(a, headers))
	a.Extra[codexMetadataRepairExtraKey] = true
	headers.Set("session-id", "different-session")
	require.NotEqual(t, enabledKey, normalizeOpenAIWSHandshakeCompatibility(a, headers))
}

func TestCodexMetadataRepairHTTPBuildersOnOff(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, passthrough := range []bool{false, true} {
			c, a := metadataRepairTestContext()
			a.Extra[codexMetadataRepairExtraKey] = enabled
			body := []byte(`{"model":"gpt-5.4","client_metadata":{"x-codex-turn-metadata":"{\"turn_id\":\"only-body\"}"}}`)
			prepareCodexMetadataRepair(c, a, body, "")
			svc := &OpenAIGatewayService{}
			var req *http.Request
			var err error
			if passthrough {
				req, err = svc.buildUpstreamRequestOpenAIPassthrough(context.Background(), c, a, body, "fake")
			} else {
				req, err = svc.buildUpstreamRequest(context.Background(), c, a, body, "fake", true, "", true)
			}
			require.NoError(t, err)
			got, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			if enabled {
				require.Equal(t, req.Header.Get(openAIWSTurnMetadataHeader), gjson.GetBytes(got, "client_metadata.x-codex-turn-metadata").String())
				require.NotEmpty(t, req.Header.Get(openAIWSTurnMetadataHeader))
			} else {
				require.Empty(t, req.Header.Get(openAIWSTurnMetadataHeader))
				require.Equal(t, string(body), string(got))
			}
		}
	}
}

func TestCodexMetadataRepairHTTPToWSOnOff(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		c, a := metadataRepairTestContext()
		a.Extra[codexMetadataRepairExtraKey] = enabled
		a.Extra["responses_websockets_v2_enabled"] = true
		metadata := `{"turn_id":"client-turn","session_id":"client-session"}`
		c.Request.Header.Set(openAIWSTurnMetadataHeader, metadata)
		c.Request.Header.Set("User-Agent", "codex_cli_rs/0.144.1")
		cfg := &config.Config{}
		cfg.Gateway.OpenAIWS.Enabled = true
		cfg.Gateway.OpenAIWS.OAuthEnabled = true
		cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
		cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 1
		cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 1
		conn := &openAIWSCaptureConn{events: [][]byte{[]byte(`{"type":"response.completed","response":{"id":"resp_repair","model":"gpt-5.4","usage":{"input_tokens":1,"output_tokens":1}}}`)}}
		dialer := &openAIWSCaptureDialer{conn: conn}
		pool := newOpenAIWSConnPool(cfg)
		pool.setClientDialerForTest(dialer)
		svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: &httpUpstreamRecorder{}, cache: &stubGatewayCache{}, openaiWSResolver: NewOpenAIWSProtocolResolver(cfg), toolCorrector: NewCodexToolCorrector(), openaiWSPool: pool}
		body, err := json.Marshal(map[string]any{"model": "gpt-5.4", "input": []any{map[string]any{"type": "message", "role": "user", "content": "test"}}, "client_metadata": map[string]any{openAIWSTurnMetadataHeader: metadata}})
		require.NoError(t, err)
		_, err = svc.Forward(context.Background(), c, a, body)
		require.NoError(t, err)
		header := dialer.lastHeaders.Get(openAIWSTurnMetadataHeader)
		payload := gjson.Get(requestToJSONString(conn.lastWrite), "client_metadata.x-codex-turn-metadata").String()
		if enabled {
			require.Empty(t, header, "每轮元数据只在 WS body 承载")
			require.Equal(t, scopeCodexAccountIdentityValue(a, 0, "turn", "client-turn"), gjson.Get(payload, "turn_id").String())
		} else {
			require.NotEqual(t, gjson.Get(header, "turn_id").String(), gjson.Get(payload, "turn_id").String(), "关闭保持原逻辑，不暗中全局修复")
		}
		pool.Close()
	}
}

func TestCodexMetadataRepairCompatibilityPaths(t *testing.T) {
	for _, route := range []string{"responses", "passthrough", "chat", "messages"} {
		for _, enabled := range []bool{false, true} {
			t.Run(route+"/"+map[bool]string{true: "on", false: "off"}[enabled], func(t *testing.T) {
				c, a := metadataRepairTestContext()
				a.Extra[codexMetadataRepairExtraKey] = enabled
				a.Extra["openai_passthrough"] = route == "passthrough"
				a.Extra["openai_device_id"] = "configured-device"
				body := map[string]any{"model": "gpt-5.4", "stream": false, "input": []any{map[string]any{"type": "message", "role": "user", "content": "test"}}, "client_metadata": map[string]any{"extension": "keep", openAIWSTurnMetadataHeader: `{"turn_id":"body-turn"}`}}
				if route == "chat" || route == "messages" {
					delete(body, "input")
					body["messages"] = []any{map[string]any{"role": "user", "content": "test"}}
					body["max_tokens"] = 16
				}
				raw, err := json.Marshal(body)
				require.NoError(t, err)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(raw))
				upstream := &httpUpstreamRecorder{resp: openAICompatSSECompletedResponse("resp_metadata_compat", "gpt-5.4")}
				svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream, cache: &stubGatewayCache{}}
				switch route {
				case "chat":
					_, err = svc.ForwardAsChatCompletions(context.Background(), c, a, raw, "client-session", "")
				case "messages":
					_, err = svc.ForwardAsAnthropic(context.Background(), c, a, raw, "client-session", "")
				default:
					_, err = svc.Forward(context.Background(), c, a, raw)
				}
				require.NoError(t, err)
				require.NotNil(t, upstream.lastReq)
				got := upstream.lastBody
				if enabled {
					require.Equal(t, upstream.lastReq.Header.Get(openAIWSTurnMetadataHeader), gjson.GetBytes(got, "client_metadata.x-codex-turn-metadata").String())
					require.Equal(t, "keep", gjson.GetBytes(got, "client_metadata.extension").String())
					require.NotEmpty(t, gjson.GetBytes(got, "client_metadata.x-codex-installation-id").String())
					if route == "chat" || route == "messages" {
						require.Equal(t, upstream.lastReq.Header.Get("session_id"), gjson.GetBytes(got, "client_metadata.session_id").String())
					}
				} else {
					require.Empty(t, upstream.lastReq.Header.Get(openAIWSTurnMetadataHeader))
				}
			})
		}
	}
}

// 两轮真实本机 WS 入站，内存假上游；覆盖握手旧值覆盖、HTTP bridge 和直接 relay。
// relay 的读取和写入并行，假上游必须等待实际收到请求后才返回对应终态。
type metadataRepairTurnConn struct {
	*openAIWSCaptureConn
	ready chan struct{}
}

func (c *metadataRepairTurnConn) WriteJSON(ctx context.Context, value any) error {
	if err := c.openAIWSCaptureConn.WriteJSON(ctx, value); err != nil {
		return err
	}
	select {
	case c.ready <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *metadataRepairTurnConn) WriteFrame(ctx context.Context, _ coderws.MessageType, payload []byte) error {
	return c.WriteJSON(ctx, json.RawMessage(payload))
}

func (c *metadataRepairTurnConn) ReadMessage(ctx context.Context) ([]byte, error) {
	select {
	case <-c.ready:
		return c.openAIWSCaptureConn.ReadMessage(ctx)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *metadataRepairTurnConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	payload, err := c.ReadMessage(ctx)
	return coderws.MessageText, payload, err
}

type metadataRepairTurnDialer struct{ conn *metadataRepairTurnConn }

func (d *metadataRepairTurnDialer) Dial(_ context.Context, _ string, _ http.Header, _ string, _ *tlsfingerprint.Profile) (openAIWSClientConn, int, http.Header, error) {
	return d.conn, 0, nil, nil
}

func TestCodexMetadataRepairWSIngressTurns(t *testing.T) {
	for _, mode := range []string{OpenAIWSIngressModeCtxPool, OpenAIWSIngressModePassthrough, OpenAIWSIngressModeHTTPBridge} {
		t.Run(mode, func(t *testing.T) {
			_, account := metadataRepairTestContext()
			account.Extra["openai_oauth_responses_websockets_v2_mode"] = mode
			cfg := newOpenAIWSV2TestConfig()
			cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
			cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 1
			cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
			cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 1
			cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
			conn := &openAIWSCaptureConn{events: [][]byte{
				[]byte(`{"type":"response.completed","response":{"id":"resp_meta_1","model":"gpt-5.4","usage":{"input_tokens":1,"output_tokens":1}}}`),
				[]byte(`{"type":"response.completed","response":{"id":"resp_meta_2","model":"gpt-5.4","usage":{"input_tokens":1,"output_tokens":1}}}`),
			}}
			dialer := &openAIWSCaptureDialer{conn: conn}
			pool := newOpenAIWSConnPool(cfg)
			defer pool.Close()
			pool.setClientDialerForTest(dialer)
			httpUpstream := &httpUpstreamRecorder{responses: []*http.Response{openAICompatSSECompletedResponse("resp_meta_1", "gpt-5.4"), openAICompatSSECompletedResponse("resp_meta_2", "gpt-5.4")}}
			svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: httpUpstream, cache: &stubGatewayCache{}, openaiWSResolver: NewOpenAIWSProtocolResolver(cfg), toolCorrector: NewCodexToolCorrector(), openaiWSPool: pool, openaiWSPassthroughDialer: dialer}
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
				c.Request.Header = r.Header.Clone()
				c.Request.Header.Set(openAIWSTurnMetadataHeader, `{"turn_id":"old-handshake","session_id":"stable-session"}`)
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
			client, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
			require.NoError(t, err)
			defer client.CloseNow()
			for _, turn := range []string{"turn-one", "turn-two"} {
				metadata, _ := json.Marshal(map[string]any{"turn_id": turn, "extra": "keep"})
				body, _ := json.Marshal(map[string]any{"type": "response.create", "model": "gpt-5.4", "input": []any{map[string]any{"type": "message", "role": "user", "content": "test"}}, "client_metadata": map[string]any{openAIWSTurnMetadataHeader: string(metadata)}})
				require.NoError(t, client.Write(ctx, coderws.MessageText, body))
				for {
					_, payload, readErr := client.Read(ctx)
					require.NoError(t, readErr)
					if gjson.GetBytes(payload, "type").String() == "response.completed" {
						break
					}
				}
			}
			_ = client.CloseNow()
			select {
			case <-errCh:
			case <-ctx.Done():
				t.Fatal("本机 WS 测试未结束")
			}
			var bodies [][]byte
			if mode == OpenAIWSIngressModeHTTPBridge {
				bodies = [][]byte{httpUpstream.lastBody}
			} else {
				for _, write := range conn.writes {
					bodies = append(bodies, []byte(requestToJSONString(write)))
				}
			}
			require.NotEmpty(t, bodies)
			last := gjson.Parse(gjson.GetBytes(bodies[len(bodies)-1], "client_metadata.x-codex-turn-metadata").String())
			require.Equal(t, scopeCodexAccountIdentityValue(account, 0, "turn", "turn-two"), last.Get("turn_id").String())
			require.Equal(t, "keep", last.Get("extra").String())
		})
	}
}
