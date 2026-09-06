package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.org/x/net/http/httpguts"
)

// 完整普通/透传 Forward 验证修复与指纹模式组合；假上游不访问生产网络。
func TestCodexMetadataRepairFingerprintForwardCompatibility(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, mode := range []string{"off", "device", "session", "full"} {
			for _, passthrough := range []bool{false, true} {
				t.Run(fmt.Sprintf("repair=%v/mode=%s/passthrough=%v", enabled, mode, passthrough), func(t *testing.T) {
					c, account := metadataRepairTestContext()
					account.Extra[codexMetadataRepairExtraKey] = enabled
					account.Extra["openai_passthrough"] = passthrough
					account.Extra[codexFingerprintModeExtraKey] = mode
					account.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
					metadata := `{"session_id":"stable-session","thread_id":"stable-thread","turn_id":"original-turn"}`
					body, err := json.Marshal(map[string]any{
						"model": "gpt-6-astra", "instructions": "Synthetic test", "stream": false,
						"input":           []any{map[string]any{"role": "user", "content": "test"}},
						"client_metadata": map[string]any{openAIWSTurnMetadataHeader: metadata},
					})
					require.NoError(t, err)
					c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
					c.Request.Header.Set("session-id", "stable-session")
					c.Request.Header.Set(openAIWSTurnMetadataHeader, metadata)
					upstream := &httpUpstreamRecorder{resp: openAICompatSSECompletedResponse("resp_compat", "gpt-6-astra")}
					svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream, cache: &stubGatewayCache{}}
					_, err = svc.Forward(context.Background(), c, account, body)
					require.NoError(t, err)
					require.NotNil(t, upstream.lastReq)
					headerTurn := gjson.Get(upstream.lastReq.Header.Get(openAIWSTurnMetadataHeader), "turn_id").String()
					bodyTurn := gjson.Get(gjson.GetBytes(upstream.lastBody, "client_metadata.x-codex-turn-metadata").String(), "turn_id").String()
					require.NotEmpty(t, bodyTurn)
					require.Equal(t, bodyTurn, headerTurn, "最终发出的头和体必须使用同一回合身份")
					require.Equal(t, enabled, account.IsCodexMetadataRepairEnabled())
					require.Equal(t, mode, string(account.GetCodexFingerprintMode()), "修复不得替用户改变指纹模式")
				})
			}
		}
	}
}

// 原始来源发生变化时不能复用旧回合快照；正常同请求重试仍复用。
func TestCodexMetadataRepairSnapshotSourceChanges(t *testing.T) {
	for _, change := range []string{"header", "cache_key", "request_kind", "tenant", "credential"} {
		t.Run(change, func(t *testing.T) {
			c, account := metadataRepairTestContext()
			body := []byte(`{"model":"gpt-6-astra","prompt_cache_key":"cache-one"}`)
			prepareCodexMetadataRepair(c, account, body, "")
			first := stagedCodexMetadataRepair(c, account)
			require.NotNil(t, first)
			prepareCodexMetadataRepair(c, account, body, "")
			require.Same(t, first, stagedCodexMetadataRepair(c, account))
			switch change {
			case "header":
				c.Request.Header.Set("session-id", "new-session")
			case "cache_key":
				body = []byte(`{"model":"gpt-6-astra","prompt_cache_key":"cache-two"}`)
			case "request_kind":
				body = []byte(`{"model":"gpt-6-astra","prompt_cache_key":"cache-one","generate":false}`)
			case "tenant":
				c.Set("api_key", &APIKey{ID: 91})
			case "credential":
				account.Credentials["chatgpt_account_id"] = "another-synthetic-account"
			}
			prepareCodexMetadataRepair(c, account, body, "")
			require.NotNil(t, stagedCodexMetadataRepair(c, account))
			require.NotSame(t, first, stagedCodexMetadataRepair(c, account), "不同来源不能复用旧快照")
		})
	}
}

// JSON 允许的控制字符不一定能进入 HTTP 头；修复必须在身份投影前拒绝。
func TestCodexMetadataRepairRejectsInvalidProjectedIdentity(t *testing.T) {
	for _, invalid := range []string{"bad\x01value", "bad\x7fvalue", "bad\r\nvalue", "bad\x00value"} {
		c, account := metadataRepairTestContext()
		raw, err := json.Marshal(map[string]any{"client_metadata": map[string]any{"session_id": invalid}})
		require.NoError(t, err)
		snapshot := buildCodexMetadataRepair(c, account, raw, "", true)
		require.Nil(t, snapshot, "不能因修复添加非法头而导致网络层拒发")
	}
	c, account := metadataRepairTestContext()
	snapshot := buildCodexMetadataRepair(c, account, []byte(`{"client_metadata":{"session_id":"valid-session"}}`), "", true)
	require.NotNil(t, snapshot)
	for _, values := range snapshot.headers {
		for _, value := range values {
			require.True(t, httpguts.ValidHeaderFieldValue(value))
		}
	}
}

// 深层扩展、长输入和嵌套同名字段不应被误改；重复投影幂等。
func TestCodexMetadataRepairRawBoundaryPreservation(t *testing.T) {
	c, account := metadataRepairTestContext()
	body := []byte(`{"input":{"client_metadata":"keep","text":"` + strings.Repeat("x", 1024*1024) + `"},"client_metadata":{"extension":{"number":9007199254740993},"session_id":"original-session"}}`)
	snapshot := buildCodexMetadataRepair(c, account, body, "", true)
	require.NotNil(t, snapshot)
	first, err := snapshot.applyRaw(body)
	require.NoError(t, err)
	second, err := snapshot.applyRaw(first)
	require.NoError(t, err)
	require.Equal(t, string(first), string(second))
	require.Equal(t, gjson.GetBytes(body, "input").Raw, gjson.GetBytes(first, "input").Raw)
	require.Equal(t, "9007199254740993", gjson.GetBytes(first, "client_metadata.extension.number").Raw)
	for _, bodyMap := range []map[string]any{{}, {"input": "keep"}} {
		snapshot.applyMap(bodyMap)
		bodyMap["client_metadata"].(map[string]any)["extension"].(map[string]any)["number"] = 0
	}
	third, err := snapshot.applyRaw(body)
	require.NoError(t, err)
	require.Equal(t, string(first), string(third), "调用方不能修改共享快照")
}

// 不完整的 OAuth 导入也不能让修复头绕过原本存在的 API Key 会话隔离。
func TestCodexMetadataRepairIncompleteOAuthTenantIsolation(t *testing.T) {
	var sessions []string
	for _, id := range []int64{21, 22} {
		c, account := metadataRepairTestContext()
		account.Credentials = map[string]any{"access_token": "synthetic-token"}
		c.Set("api_key", &APIKey{ID: id})
		body := []byte(`{"model":"gpt-6-astra","client_metadata":{"session_id":"same-client-session","thread_id":"same-client-thread"}}`)
		snapshot := buildCodexMetadataRepair(c, account, body, "", true)
		if snapshot == nil {
			// 无可靠身份时保留原流程也是安全退路，不能把未隔离的值补进头。
			continue
		}
		sessions = append(sessions, snapshot.headers.Get("session_id"))
	}
	if len(sessions) == 2 {
		require.NotEqual(t, sessions[0], sessions[1], "修复不得撤销原有跨 API Key 隔离")
	}
}

// 同账号请求内部兼容重试只清理明确被拒绝的业务字段，metadata 不重生成。
func TestCodexMetadataRepairHTTPRetryKeepsSnapshot(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		for _, mode := range []string{"off", "device", "session", "full"} {
			t.Run(fmt.Sprintf("passthrough=%v/mode=%s", passthrough, mode), func(t *testing.T) {
				c, account := metadataRepairTestContext()
				account.Extra["openai_passthrough"] = passthrough
				account.Extra[codexFingerprintModeExtraKey] = mode
				account.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
				body := []byte(`{"model":"gpt-6-astra","instructions":"Synthetic retry","stream":false,"input":[{"type":"reasoning","encrypted_content":"synthetic-encrypted","summary":[]},{"role":"user","content":"keep"}],"client_metadata":{"session_id":"stable-session"}}`)
				errorBody := `{"error":{"code":"invalid_encrypted_content","message":"Invalid encrypted reasoning content"}}`
				if passthrough {
					// 两条旧路径支持不同的兼容重试；不为测试而改变既有重试策略。
					body = []byte(`{"model":"gpt-6-astra","instructions":"Synthetic retry","stream":false,"max_output_tokens":1024,"input":[{"role":"user","content":"keep"}],"client_metadata":{"session_id":"stable-session"}}`)
					errorBody = `{"error":{"code":"unsupported_parameter","message":"Unsupported parameter: max_output_tokens","param":"max_output_tokens"}}`
				}
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
				upstream := &httpUpstreamRecorder{responses: []*http.Response{
					newOpenAIRejectedFieldTestResponse(http.StatusBadRequest, errorBody),
					openAICompatSSECompletedResponse("resp_retry", "gpt-6-astra"),
				}}
				svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream, cache: &stubGatewayCache{}}
				_, err := svc.Forward(context.Background(), c, account, body)
				require.NoError(t, err)
				require.Len(t, upstream.bodies, 2)
				if passthrough {
					require.True(t, gjson.GetBytes(upstream.bodies[0], "max_output_tokens").Exists())
					require.False(t, gjson.GetBytes(upstream.bodies[1], "max_output_tokens").Exists())
				} else {
					require.Contains(t, string(upstream.bodies[0]), "synthetic-encrypted")
					require.NotContains(t, string(upstream.bodies[1]), "synthetic-encrypted")
				}
				require.Equal(t, gjson.GetBytes(upstream.bodies[0], "client_metadata").Raw, gjson.GetBytes(upstream.bodies[1], "client_metadata").Raw)
				require.Equal(t, upstream.requests[0].Header.Get(openAIWSTurnMetadataHeader), upstream.requests[1].Header.Get(openAIWSTurnMetadataHeader))
				require.Contains(t, gjson.GetBytes(upstream.bodies[1], "input").Raw, "keep")
			})
		}
	}
}

// OAuth/AT、换账号、影子母账号与租户边界仍由原身份规则决定，不从上一尝试串值。
func TestCodexMetadataRepairCredentialAndTenantMatrix(t *testing.T) {
	c, account := metadataRepairTestContext()
	body := []byte(`{"client_metadata":{"session_id":"shared-client-session","thread_id":"shared-client-thread","turn_id":"shared-client-turn"}}`)
	prepareCodexMetadataRepair(c, account, body, "")
	first := stagedCodexMetadataRepair(c, account)
	require.NotNil(t, first)
	other := *account
	other.ID = 43
	other.Credentials = map[string]any{"access_token": "another-synthetic-token", "chatgpt_account_id": "another-synthetic-account"}
	prepareCodexMetadataRepair(c, &other, body, "")
	second := stagedCodexMetadataRepair(c, &other)
	require.NotNil(t, second)
	require.Nil(t, stagedCodexMetadataRepair(c, account))
	require.NotEqual(t, first.metadata["session_id"], second.metadata["session_id"])
	prepareCodexMetadataRepair(c, account, body, "")
	require.Equal(t, first.metadata["session_id"], stagedCodexMetadataRepair(c, account).metadata["session_id"])
	c.Set("api_key", &APIKey{ID: 123})
	prepareCodexMetadataRepair(c, account, body, "")
	require.NotEqual(t, first.metadata["session_id"], stagedCodexMetadataRepair(c, account).metadata["session_id"])
	for _, accountType := range []string{AccountTypeAPIKey, AccountTypeSetupToken, AccountTypeOAuth} {
		copyAccount := other
		copyAccount.Type = accountType
		prepareCodexMetadataRepair(c, &copyAccount, body, "")
		if accountType == AccountTypeAPIKey {
			require.Nil(t, stagedCodexMetadataRepair(c, &copyAccount))
		} else {
			require.NotNil(t, stagedCodexMetadataRepair(c, &copyAccount))
		}
	}
	parent := *account
	shadow := *account
	shadow.ID = 99
	shadow.ParentAccountID = &parent.ID
	shadow.Credentials = nil
	svc := &OpenAIGatewayService{accountRepo: &codexAccountIdentityRepoStub{account: &parent}}
	_, err := svc.prepareCodexAccountIdentitySource(context.Background(), c, &shadow)
	require.NoError(t, err)
	prepareCodexMetadataRepair(c, &shadow, body, "")
	shadowSnapshot := stagedCodexMetadataRepair(c, &shadow)
	require.NotNil(t, shadowSnapshot)
	_, err = svc.prepareCodexAccountIdentitySource(context.Background(), c, &parent)
	require.NoError(t, err)
	prepareCodexMetadataRepair(c, &parent, body, "")
	require.Equal(t, shadowSnapshot.metadata["session_id"], stagedCodexMetadataRepair(c, &parent).metadata["session_id"])
}

// 修复开关不接管旧 Compact/API Key 路径；即使 context 中已有快照也不能泄漏。
func TestCodexMetadataRepairExcludedOutboundPaths(t *testing.T) {
	for _, variant := range []string{"compact", "apikey", "disabled", "invalid"} {
		t.Run(variant, func(t *testing.T) {
			c, account := metadataRepairTestContext()
			prepareCodexMetadataRepair(c, account, []byte(`{}`), "old-session")
			body := []byte(`{"model":"gpt-6-astra","input":"keep","client_metadata":{"extension":"keep"}}`)
			switch variant {
			case "compact":
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", bytes.NewReader(body))
			case "apikey":
				account.Type = AccountTypeAPIKey
			case "disabled":
				account.Extra[codexMetadataRepairExtraKey] = false
			case "invalid":
				body = []byte(`{"model":"gpt-6-astra","input":"keep","client_metadata":[]}`)
			}
			prepareCodexMetadataRepair(c, account, body, "")
			require.Nil(t, stagedCodexMetadataRepair(c, account))
			svc := &OpenAIGatewayService{cfg: &config.Config{}}
			req, err := svc.buildUpstreamRequest(context.Background(), c, account, body, "synthetic-token", false, "", false)
			require.NoError(t, err)
			out, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Equal(t, string(body), string(out))
			require.Empty(t, req.Header.Get(openAIWSTurnMetadataHeader))
		})
	}
}

// 并行投影只读取同一快照，每个调用方获得独立对象，不污染其它 WS 回合。
func TestCodexMetadataRepairConcurrentProjection(t *testing.T) {
	c, account := metadataRepairTestContext()
	snapshot := buildCodexMetadataRepair(c, account, []byte(`{"client_metadata":{"extension":{"value":"keep"}}}`), "session", true)
	require.NotNil(t, snapshot)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := map[string]any{}
			snapshot.applyMap(body)
			body["client_metadata"].(map[string]any)["extension"].(map[string]any)["value"] = "mutated"
			headers := make(http.Header)
			snapshot.applyHeaders(headers)
		}()
	}
	wg.Wait()
	require.Equal(t, "keep", snapshot.metadata["extension"].(map[string]any)["value"])
}

// 已知同账号同线程时，改变请求追踪号不应破坏原连接续接。
func TestCodexMetadataRepairPoolRequestTraceCompatibility(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("repair=%v/strict=%v", enabled, strict), func(t *testing.T) {
				c, account := metadataRepairTestContext()
				account.Extra[codexMetadataRepairExtraKey] = enabled
				cfg := newOpenAIWSV2TestConfig()
				cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 1
				cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
				cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 1
				pool := newOpenAIWSConnPool(cfg)
				defer pool.Close()
				dialer := &openAIWSCountingDialer{}
				pool.setClientDialerForTest(dialer)
				svc := &OpenAIGatewayService{cfg: cfg}
				body := []byte(`{"model":"gpt-6-astra","client_metadata":{"session_id":"stable-session","thread_id":"stable-thread"}}`)
				makeHeaders := func(requestID string) http.Header {
					c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
					c.Request.Header.Set("session-id", "stable-session")
					c.Request.Header.Set("x-client-request-id", requestID)
					prepareCodexMetadataRepair(c, account, body, "")
					headers, _, err := svc.buildOpenAIWSHeaders(context.Background(), c, account, "synthetic-token", OpenAIWSProtocolDecision{}, true, "", "", "", "gpt-6-astra", "")
					require.NoError(t, err)
					return headers
				}
				first, err := pool.Acquire(context.Background(), openAIWSAcquireRequest{Account: account, WSURL: "wss://synthetic.invalid/responses", Headers: makeHeaders("request-one")})
				require.NoError(t, err)
				connID := first.ConnID()
				first.Release()
				second, err := pool.Acquire(context.Background(), openAIWSAcquireRequest{Account: account, WSURL: "wss://synthetic.invalid/responses", Headers: makeHeaders("request-two"), PreferredConnID: connID, ForcePreferredConn: strict})
				require.NoError(t, err)
				defer second.Release()
				require.Equal(t, connID, second.ConnID())
				require.Equal(t, 1, dialer.DialCount())
			})
		}
	}
}

// 只消除重复的追踪字段约束；真实身份变化、未知线程及关闭分支仍保持隔离。
func TestCodexMetadataRepairPoolPreservesIdentityBoundaries(t *testing.T) {
	_, account := metadataRepairTestContext()
	headers := make(http.Header)
	headers.Set("x-codex-installation-id", "device-a")
	headers.Set("session-id", "session-a")
	headers.Set("session_id", "session-a")
	headers.Set("thread-id", "thread-a")
	headers.Set("x-client-request-id", "trace-a")
	headers.Set("x-codex-window-id", "window-a")
	base := normalizeOpenAIWSHandshakeCompatibility(account, headers)
	for _, field := range []string{"x-codex-installation-id", "session-id", "session_id", "thread-id", "x-codex-window-id", "x-codex-beta-features"} {
		changed := headers.Clone()
		changed.Set(field, "different")
		require.NotEqual(t, base, normalizeOpenAIWSHandshakeCompatibility(account, changed), field)
	}
	headers.Del("thread-id")
	base = normalizeOpenAIWSHandshakeCompatibility(account, headers)
	headers.Set("x-client-request-id", "trace-b")
	require.NotEqual(t, base, normalizeOpenAIWSHandshakeCompatibility(account, headers), "未知线程必须保留追踪头的隔离兜底")
	account.Extra[codexMetadataRepairExtraKey] = false
	account.Extra[codexFingerprintModeExtraKey] = "session"
	account.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
	headers.Set("thread-id", "thread-a")
	base = normalizeOpenAIWSHandshakeCompatibility(account, headers)
	headers.Set("x-client-request-id", "trace-c")
	require.NotEqual(t, base, normalizeOpenAIWSHandshakeCompatibility(account, headers), "关闭修复时仍按原指纹策略检查")
	account.Extra[codexMetadataRepairExtraKey] = true
	for _, mode := range []string{"session", "full"} {
		account.Extra[codexFingerprintModeExtraKey] = mode
		base = normalizeOpenAIWSHandshakeCompatibility(account, headers)
		changed := headers.Clone()
		changed.Set("x-client-request-id", "another-tenant")
		require.NotEqual(t, base, normalizeOpenAIWSHandshakeCompatibility(account, changed), "合并线程时不能取消追踪头隔离："+mode)
	}
}

// 对照开关前后真正发出的 body，除 client_metadata 外的业务字段必须相同。
// 允许既有协议转换继续工作，但不能由修复开关额外改变输入、工具、模型或预算。
func TestCodexMetadataRepairPreservesBusinessPayload(t *testing.T) {
	for _, route := range []string{"responses", "passthrough", "chat", "messages"} {
		for _, mode := range []string{"off", "device", "session", "full"} {
			t.Run(route+"/"+mode, func(t *testing.T) {
				var originalWire []byte
				var originalHeaders http.Header
				for _, enabled := range []bool{false, true} {
					c, account := metadataRepairTestContext()
					account.Extra[codexMetadataRepairExtraKey] = enabled
					account.Extra[codexFingerprintModeExtraKey] = mode
					account.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
					account.Extra["openai_passthrough"] = route == "passthrough"
					body := map[string]any{
						"model": "gpt-6-astra", "instructions": "Keep user instructions", "stream": false,
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
