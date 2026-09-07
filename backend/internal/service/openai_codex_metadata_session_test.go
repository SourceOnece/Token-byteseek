package service

import (
	"context"
	"encoding/json"
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

// 同一连接缺省身份可继承；显式新线程必须隔离且不能携带前线程的父/代理关系。
func TestCodexMetadataRepairSessionDefaultsAndBoundaries(t *testing.T) {
	for _, mode := range []string{"off", "device", "session", "full"} {
		t.Run(mode, func(t *testing.T) {
			c, a := metadataRepairTestContext()
			c.Set("api_key", &APIKey{ID: 83})
			a.Extra[codexFingerprintModeExtraKey] = mode
			a.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
			c.Request.Header.Set(openAISubagentHeader, "collab_spawn")
			c.Request.Header.Set(codexParentThreadIDHeader, "parent")
			state := &codexMetadataSession{}
			first := state.build(c, a, []byte(`{"client_metadata":{"session_id":"s","thread_id":"t","x-codex-window-id":"w","x-codex-turn-metadata":"{\"turn_id\":\"one\",\"parent_turn_id\":\"parent-turn\",\"root_turn_id\":\"root\",\"sandbox\":\"real\"}"}}`), true)
			require.NotNil(t, first)
			next := state.build(c, a, []byte(`{"client_metadata":{"turn_id":"two"}}`), false)
			require.NotNil(t, next)
			for _, field := range []string{"session_id", "thread-id", "x-codex-window-id", codexParentThreadIDHeader, openAISubagentHeader} {
				require.Equal(t, first.headers.Get(field), next.headers.Get(field), field)
			}
			require.NotEqual(t, first.headers.Get("turn-id"), next.headers.Get("turn-id"))
			nextTurn := gjson.Parse(next.metadata[openAIWSTurnMetadataHeader].(string))
			for _, field := range []string{"parent_turn_id", "root_turn_id", "sandbox"} {
				require.False(t, nextTurn.Get(field).Exists(), field)
			}
			changed := state.build(c, a, []byte(`{"client_metadata":{"thread_id":"different","turn_id":"three"}}`), false)
			require.NotNil(t, changed)
			require.NotEqual(t, first.headers.Get("thread-id"), changed.headers.Get("thread-id"))
			require.Empty(t, changed.headers.Get(codexParentThreadIDHeader))
			require.Empty(t, changed.headers.Get(openAISubagentHeader))
			require.Equal(t, first.headers.Get("session_id"), changed.headers.Get("session_id"))
			c.Set("api_key", &APIKey{ID: 84})
			tenant := state.build(c, a, []byte(`{"client_metadata":{"turn_id":"four"}}`), false)
			require.NotNil(t, tenant)
			require.NotEqual(t, first.headers.Get("thread-id"), tenant.headers.Get("thread-id"))
			a.Extra[codexMetadataRepairExtraKey] = false
			require.Nil(t, state.build(c, a, []byte(`{}`), false))
			require.Nil(t, state.stable)
		})
	}
}

// 主身份不是字符串时不输出相互矛盾的新快照；body 请求类型不受旧握手种类污染。
func TestCodexMetadataRepairPrimaryTypesAndCurrentKind(t *testing.T) {
	c, a := metadataRepairTestContext()
	for _, value := range []any{7, map[string]any{"bad": true}, []any{"bad"}} {
		for _, field := range []string{"installation_id", "session_id", "thread_id", "turn_id", "window_id"} {
			nested, _ := json.Marshal(map[string]any{field: value})
			body, _ := json.Marshal(map[string]any{"client_metadata": map[string]any{openAIWSTurnMetadataHeader: string(nested)}})
			require.Nil(t, buildCodexMetadataRepair(c, a, body, "", true), field)
		}
	}
	c.Request.Header.Set(openAIWSTurnMetadataHeader, `{"request_kind":"memory","turn_id":"stale","turn_started_at_unix_ms":1}`)
	s := buildCodexMetadataRepair(c, a, []byte(`{"input":"hello","client_metadata":{"x-codex-turn-metadata":"{\"session_id\":\"s\",\"thread_id\":\"t\",\"turn_id\":\"current\"}"}}`), "", true)
	require.NotNil(t, s)
	turn := gjson.Parse(s.metadata[openAIWSTurnMetadataHeader].(string))
	require.Equal(t, "turn", turn.Get("request_kind").String())
	require.True(t, turn.Get("thread_id").Exists())
	require.NotEqual(t, int64(1), turn.Get("turn_started_at_unix_ms").Int())
}

// 更新帧仅统一实际存在的身份，不把 session.update 改成用户回合。
func TestCodexMetadataRepairSessionUpdateNoSyntheticTurn(t *testing.T) {
	for _, mode := range []string{"off", "device", "session", "full"} {
		c, a := metadataRepairTestContext()
		a.Extra[codexFingerprintModeExtraKey] = mode
		a.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
		state := &codexMetadataSession{}
		first := state.build(c, a, []byte(`{"client_metadata":{"session_id":"s","thread_id":"t","turn_id":"one"}}`), true)
		update := state.build(c, a, []byte(`{"type":"session.update","client_metadata":{"session_id":"s","extension":"keep"}}`), false)
		require.NotNil(t, update)
		require.NotNil(t, first)
		require.NotContains(t, update.metadata, "turn_id")
		require.NotContains(t, update.metadata, openAIWSTurnMetadataHeader)
		require.Equal(t, "keep", update.metadata["extension"])
		require.Equal(t, first.metadata["session_id"], update.metadata["session_id"])
		require.NotContains(t, update.metadata, "thread_id", "更新对象未声明线程，不扩大更新范围")
		next := state.build(c, a, []byte(`{"client_metadata":{"turn_id":"two"}}`), false)
		require.Equal(t, first.metadata["thread_id"], next.metadata["thread_id"])
	}
}

// 真实连接池首轮带官方兼容头，后续省略身份仍能严格获取同一连接。
func TestCodexMetadataRepairSessionStrictPoolReuse(t *testing.T) {
	c, a := metadataRepairTestContext()
	cfg := newOpenAIWSV2TestConfig()
	cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 1
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 1
	pool := newOpenAIWSConnPool(cfg)
	defer pool.Close()
	dialer := &openAIWSCountingDialer{}
	pool.setClientDialerForTest(dialer)
	svc := &OpenAIGatewayService{cfg: cfg}
	state := &codexMetadataSession{}
	bodies := [][]byte{[]byte(`{"client_metadata":{"session_id":"s","thread_id":"t","turn_id":"one","x-openai-subagent":"collab_spawn","x-codex-parent-thread-id":"parent"}}`), []byte(`{"client_metadata":{"turn_id":"two"}}`)}
	var id string
	for i, body := range bodies {
		c.Set(codexMetadataRepairContextKey, state.build(c, a, body, i == 0))
		h, _, err := svc.buildOpenAIWSHeaders(context.Background(), c, a, "synthetic", OpenAIWSProtocolDecision{}, true, "", "", "", "gpt-6-astra", "")
		require.NoError(t, err)
		require.NotEmpty(t, h.Get(openAIWSTurnMetadataHeader))
		require.Equal(t, "collab_spawn", h.Get(openAISubagentHeader))
		lease, err := pool.Acquire(context.Background(), openAIWSAcquireRequest{Account: a, WSURL: "wss://synthetic.invalid", Headers: h, PreferredConnID: id, ForcePreferredConn: i > 0})
		require.NoError(t, err)
		if i == 0 {
			id = lease.ConnID()
		} else {
			require.Equal(t, id, lease.ConnID())
		}
		lease.Release()
	}
	require.Equal(t, 1, dialer.DialCount())
}

// 已知官方环境/扩展、大整数和非 ASCII 完整保留；工具库存依官方仅在 body。
func TestCodexMetadataRepairOfficialInventory(t *testing.T) {
	c, a := metadataRepairTestContext()
	fields := map[string]any{
		"installation_id": "i", "session_id": "s", "thread_id": "t", "agent_name": "agent", "turn_id": "turn", "window_id": "w", "window_number": json.Number("9007199254740993"), "context_window_id": "context",
		"request_kind": "turn", "forked_from_thread_id": "fork", "forked_from_ordinal_exclusive": json.Number("9007199254740993"), "parent_thread_id": "parent", "parent_turn_id": "pt", "root_turn_id": "rt",
		"subagent_kind": "thread_spawn", "thread_source": "cli", "turn_trigger": "user", "sandbox": "真实环境", "sandbox_mode": "workspace-write", "auto_review_enabled": false, "node_repl_auto_review_required": false, "node_repl_disabled": false,
		"workspaces": map[string]any{"/synthetic": map[string]any{"has_changes": true}}, "tool_namespaces_info": map[string]any{"test": map[string]any{}}, "turn_started_at_unix_ms": json.Number("1788740827000"), "history_ingest_requested": false, "compaction": map[string]any{"phase": "test"}, "extra.test": "中文🚲",
	}
	blob, _ := json.Marshal(fields)
	flat := map[string]any{openAIWSTurnMetadataHeader: string(blob), "ws_request_header_traceparent": "00-synthetic", "ws_request_header_tracestate": "synthetic", "x-codex-ws-stream-request-start-ms": "1788740827000", "ws_request_header_x_openai_internal_codex_responses_lite": "true"}
	body, _ := json.Marshal(map[string]any{"client_metadata": flat})
	s := buildCodexMetadataRepair(c, a, body, "", true)
	require.NotNil(t, s)
	got := gjson.Parse(s.metadata[openAIWSTurnMetadataHeader].(string))
	for field := range fields {
		require.Contains(t, got.Map(), field)
	}
	require.Equal(t, "9007199254740993", got.Get("window_number").Raw)
	require.Equal(t, "中文🚲", got.Get("extra\\.test").String())
	require.False(t, gjson.Get(s.headers.Get(openAIWSTurnMetadataHeader), "tool_namespaces_info").Exists())
	c.Request.Header.Set("x-openai-memgen-request", "true")
	s = buildCodexMetadataRepair(c, a, body, "", true)
	require.Equal(t, "true", s.headers.Get("x-openai-memgen-request"))
	require.Equal(t, "collab_spawn", s.headers.Get(openAISubagentHeader))
	for key, value := range flat {
		if key != openAIWSTurnMetadataHeader {
			require.Equal(t, value, s.metadata[key])
		}
	}
}

// 兼容头与 body 可证明属于同一回合时补入遗漏扩展，不丢有来源的 Metadata。
func TestCodexMetadataRepairSameTurnHeaderComplementsBody(t *testing.T) {
	c, a := metadataRepairTestContext()
	c.Request.Header.Set(openAIWSTurnMetadataHeader, `{"turn_id":"current","sandbox":"actual","request_kind":"turn","turn_started_at_unix_ms":123}`)
	s := buildCodexMetadataRepair(c, a, []byte(`{"client_metadata":{"x-codex-turn-metadata":"{\"turn_id\":\"current\",\"thread_id\":\"thread\"}"}}`), "", true)
	require.NotNil(t, s)
	turn := gjson.Parse(s.metadata[openAIWSTurnMetadataHeader].(string))
	require.Equal(t, "actual", turn.Get("sandbox").String())
	require.Equal(t, int64(123), turn.Get("turn_started_at_unix_ms").Int())
}

// 运维观测只给当前入站附加 context，不能让同账号重试生成不同回合号。
func TestCodexMetadataRepairRetryAfterRequestWithContext(t *testing.T) {
	c, a := metadataRepairTestContext()
	body := []byte(`{"model":"gpt-6-astra","input":"same"}`)
	prepareCodexMetadataRepair(c, a, body, "")
	first := stagedCodexMetadataRepair(c, a)
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), struct{}{}, "synthetic"))
	prepareCodexMetadataRepair(c, a, body, "")
	require.Same(t, first, stagedCodexMetadataRepair(c, a))
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	prepareCodexMetadataRepair(c, a, body, "")
	require.NotEqual(t, first.metadata["turn_id"], stagedCodexMetadataRepair(c, a).metadata["turn_id"])
}

// 显式清空与省略不同，清空后不能从旧握手/指纹/session 再生成该字段。
func TestCodexMetadataRepairExplicitIdentityClear(t *testing.T) {
	for _, mode := range []string{"off", "device", "session", "full"} {
		c, a := metadataRepairTestContext()
		a.Extra[codexFingerprintModeExtraKey] = mode
		a.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
		c.Request.Header.Set("session-id", "s")
		c.Request.Header.Set("thread-id", "old")
		state := &codexMetadataSession{}
		require.NotNil(t, state.build(c, a, []byte(`{"client_metadata":{"thread_id":"old","turn_id":"one"}}`), true))
		cleared := state.build(c, a, []byte(`{"client_metadata":{"thread_id":null,"turn_id":"two"}}`), false)
		require.NotNil(t, cleared)
		require.NotContains(t, cleared.metadata, "thread_id")
		require.Empty(t, cleared.headers.Get("thread-id"))
		require.False(t, gjson.Get(cleared.metadata[openAIWSTurnMetadataHeader].(string), "thread_id").Exists())
		omitted := state.build(c, a, []byte(`{"client_metadata":{"turn_id":"three"}}`), false)
		require.NotNil(t, omitted)
		require.NotContains(t, omitted.metadata, "thread_id")
	}
}

// 显式清空子任务/父线程后，省略字段的下一帧也不能从旧握手恢复。
func TestCodexMetadataRepairStableLineageClear(t *testing.T) {
	c, a := metadataRepairTestContext()
	c.Request.Header.Set(openAISubagentHeader, "collab_spawn")
	c.Request.Header.Set(codexParentThreadIDHeader, "old-parent")
	state := &codexMetadataSession{}
	require.NotNil(t, state.build(c, a, []byte(`{"client_metadata":{"thread_id":"t","turn_id":"one"}}`), true))
	cleared := state.build(c, a, []byte(`{"client_metadata":{"x-openai-subagent":null,"parent_thread_id":null,"turn_id":"two"}}`), false)
	require.NotNil(t, cleared)
	require.Empty(t, cleared.headers.Get(openAISubagentHeader))
	require.Empty(t, cleared.headers.Get(codexParentThreadIDHeader))
	next := state.build(c, a, []byte(`{"client_metadata":{"turn_id":"three"}}`), false)
	require.NotNil(t, next)
	require.Empty(t, next.headers.Get(openAISubagentHeader))
	require.Empty(t, next.headers.Get(codexParentThreadIDHeader))
}

// 相同自定义 kind 在 flat 再次声明也不能清除对应的真实兼容头。
func TestCodexMetadataRepairFlatKindKeepsMatchingHeader(t *testing.T) {
	c, a := metadataRepairTestContext()
	state := &codexMetadataSession{}
	first := state.build(c, a, []byte(`{"client_metadata":{"thread_id":"t","subagent_kind":"custom-kind","x-openai-subagent":"custom-header","turn_id":"one"}}`), true)
	require.NotNil(t, first)
	next := state.build(c, a, []byte(`{"client_metadata":{"subagent_kind":"custom-kind","turn_id":"two"}}`), false)
	require.NotNil(t, next)
	require.Equal(t, "custom-header", next.headers.Get(openAISubagentHeader))
}

// Memory 不是上一普通线程的子回合，不能自动继承该线程的父/代理来源，之后普通会话仍保留。
func TestCodexMetadataRepairMemoryDoesNotBorrowStableLineage(t *testing.T) {
	c, a := metadataRepairTestContext()
	state := &codexMetadataSession{}
	first := state.build(c, a, []byte(`{"client_metadata":{"session_id":"s","thread_id":"t","turn_id":"one","x-codex-parent-thread-id":"parent","x-openai-subagent":"collab_spawn"}}`), true)
	memory := state.build(c, a, []byte(`{"client_metadata":{"x-codex-turn-metadata":"{\"request_kind\":\"memory\",\"turn_id\":\"memory\",\"root_turn_id\":\"memory\"}"}}`), false)
	require.NotNil(t, memory)
	require.Empty(t, memory.headers.Get(codexParentThreadIDHeader))
	require.Empty(t, memory.headers.Get(openAISubagentHeader))
	next := state.build(c, a, []byte(`{"client_metadata":{"turn_id":"next"}}`), false)
	require.Equal(t, first.headers.Get("thread-id"), next.headers.Get("thread-id"))
	require.Equal(t, first.headers.Get(codexParentThreadIDHeader), next.headers.Get(codexParentThreadIDHeader))
}

// 当前轮补缺后的原始身份参与原有安全重放，新账号重新隔离，不能二次隔离旧账号编号。
func TestCodexMetadataRepairReplaySourceAccountSwitch(t *testing.T) {
	c, a := metadataRepairTestContext()
	state := &codexMetadataSession{}
	first := state.build(c, a, []byte(`{"client_metadata":{"session_id":"s","thread_id":"t","turn_id":"one"}}`), true)
	body := []byte(`{"input":[{"role":"user","content":"unchanged"}],"prompt_cache_key":"cache","client_metadata":{"turn_id":"two"}}`)
	next := state.build(c, a, body, false)
	require.NotNil(t, next)
	stageCodexMetadataReplay(c, body, state.stable)
	c.Request = c.Request.Clone(c.Request.Context()) // 旧链路合法替换 request 对象仍属同一个入站 WS。
	other := *a
	other.ID++
	other.Credentials = map[string]any{"chatgpt_account_id": "other", "access_token": "synthetic"}
	switched := newCodexMetadataSession(c, &other, body).build(c, &other, body, true)
	require.NotNil(t, switched)
	require.Equal(t, scopeCodexAccountIdentityValue(&other, 0, "thread", "t"), switched.metadata["thread_id"])
	require.NotEqual(t, first.metadata["thread_id"], switched.metadata["thread_id"])
	other.Extra = map[string]any{codexMetadataRepairExtraKey: false}
	stageCodexMetadataReplay(c, body, state.stable)
	disabled := newCodexMetadataSession(c, &other, body)
	require.Nil(t, disabled.stable)
	require.Nil(t, disabled.build(c, &other, body, true))
	require.False(t, gjson.GetBytes(body, "client_metadata.thread_id").Exists(), "不得修改原重放 payload")
	require.Equal(t, "cache", gjson.GetBytes(body, "prompt_cache_key").String())
	stageCodexMetadataReplay(c, body, state.stable)
	a.Extra[codexMetadataRepairExtraKey] = true
	wrong := newCodexMetadataSession(c, a, []byte(`{"client_metadata":{"turn_id":"unrelated"}}`))
	require.Nil(t, wrong.stable, "不同 payload 不能消费旧回合默认值")
	require.Nil(t, newCodexMetadataSession(c, a, body).stable, "重放默认值只能消费一次")
}

// 实际透传 relay 的三帧链路：create → session.update → create，不能只测 helper。
func TestCodexMetadataRepairSessionUpdateRelay(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{true: "on", false: "off"}[enabled], func(t *testing.T) {
			_, a := metadataRepairTestContext()
			a.Extra[codexMetadataRepairExtraKey] = enabled
			a.Extra[codexFingerprintModeExtraKey] = "full"
			a.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
			a.Extra["openai_oauth_responses_websockets_v2_mode"] = OpenAIWSIngressModePassthrough
			cfg := newOpenAIWSV2TestConfig()
			cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
			conn := &openAIWSCaptureConn{events: [][]byte{
				[]byte(`{"type":"response.completed","response":{"id":"resp_one","model":"gpt-6-astra","usage":{"input_tokens":1,"output_tokens":1}}}`),
				[]byte(`{"type":"session.updated","session":{}}`),
				[]byte(`{"type":"response.completed","response":{"id":"resp_two","model":"gpt-6-astra","usage":{"input_tokens":1,"output_tokens":1}}}`),
			}}
			dialer := &metadataRepairTurnDialer{conn: &metadataRepairTurnConn{openAIWSCaptureConn: conn, ready: make(chan struct{}, 3)}}
			svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: &httpUpstreamRecorder{}, cache: &stubGatewayCache{}, openaiWSResolver: NewOpenAIWSProtocolResolver(cfg), toolCorrector: NewCodexToolCorrector(), openaiWSPassthroughDialer: dialer}
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
				errCh <- svc.ProxyResponsesWebSocketFromClient(r.Context(), c, client, a, "synthetic-token", first, nil)
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			client, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
			require.NoError(t, err)
			defer client.CloseNow()
			payloads := []string{
				`{"type":"response.create","model":"gpt-6-astra","input":"one","client_metadata":{"session_id":"s","thread_id":"t","turn_id":"one"}}`,
				`{"type":"session.update","session":{"instructions":"keep"},"client_metadata":{"session_id":"s","extension":"keep"}}`,
				`{"type":"response.create","model":"gpt-6-astra","input":"two","client_metadata":{"turn_id":"two"}}`,
			}
			for i, payload := range payloads {
				require.NoError(t, client.Write(ctx, coderws.MessageText, []byte(payload)))
				want := "response.completed"
				if i == 1 {
					want = "session.updated"
				}
				for {
					_, event, err := client.Read(ctx)
					require.NoError(t, err)
					if gjson.GetBytes(event, "type").String() == want {
						break
					}
				}
			}
			_ = client.CloseNow()
			select {
			case <-errCh:
			case <-ctx.Done():
				t.Fatal("relay 未结束")
			}
			require.Len(t, conn.writes, 3)
			update := gjson.Parse(requestToJSONString(conn.writes[1]))
			require.Equal(t, "session.update", update.Get("type").String())
			require.Equal(t, "keep", update.Get("session.instructions").String())
			require.Equal(t, "keep", update.Get("client_metadata.extension").String())
			require.False(t, update.Get("client_metadata.turn_id").Exists())
			require.False(t, update.Get("client_metadata.thread_id").Exists())
			if enabled {
				first := gjson.Parse(requestToJSONString(conn.writes[0]))
				next := gjson.Parse(requestToJSONString(conn.writes[2]))
				require.Equal(t, first.Get("client_metadata.session_id").String(), update.Get("client_metadata.session_id").String())
				require.Equal(t, first.Get("client_metadata.thread_id").String(), next.Get("client_metadata.thread_id").String())
			}
		})
	}
}
