package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 所有入口仅在开启时投影两个新增兼容头；关闭路径保留已有过滤行为。
func TestCodexMetadataRepairLineageHeaders(t *testing.T) {
	for _, route := range []string{"http", "passthrough", "ws", "chat", "messages"} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/enabled=%v", route, enabled), func(t *testing.T) {
				c, account := metadataRepairTestContext()
				account.Extra[codexMetadataRepairExtraKey] = enabled
				c.Set("api_key", &APIKey{ID: 73})
				c.Request.Header.Set(codexParentThreadIDHeader, "parent-thread")
				c.Request.Header.Set(openAISubagentHeader, "collab_spawn")
				body := []byte(`{"model":"gpt-6-astra","input":"test","messages":[{"role":"user","content":"test"}],"max_tokens":64}`)
				prepareCodexMetadataRepair(c, account, body, "")
				upstream := &httpUpstreamRecorder{resp: openAICompatSSECompletedResponse("resp_lineage", "gpt-6-astra")}
				svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream, cache: &stubGatewayCache{}}
				var headers http.Header
				var err error
				switch route {
				case "ws":
					headers, _, err = svc.buildOpenAIWSHeaders(context.Background(), c, account, "synthetic-token", OpenAIWSProtocolDecision{}, true, "", "", "", "gpt-6-astra", "")
				case "chat":
					_, err = svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "")
					require.NotNil(t, upstream.lastReq)
					headers = upstream.lastReq.Header
				case "messages":
					_, err = svc.ForwardAsAnthropic(context.Background(), c, account, body, "", "")
					require.NotNil(t, upstream.lastReq)
					headers = upstream.lastReq.Header
				default:
					var req *http.Request
					if route == "http" {
						req, err = svc.buildUpstreamRequest(context.Background(), c, account, body, "synthetic-token", true, "", true)
					} else {
						req, err = svc.buildUpstreamRequestOpenAIPassthrough(context.Background(), c, account, body, "synthetic-token")
					}
					require.NoError(t, err)
					defer req.Body.Close()
					headers = req.Header
				}
				require.NoError(t, err)
				if !enabled {
					require.Empty(t, headers.Get(codexParentThreadIDHeader))
					require.Empty(t, headers.Get(openAISubagentHeader))
					return
				}
				want := scopeCodexAccountIdentityValue(account, 73, "thread", "parent-thread")
				require.Equal(t, want, headers.Get(codexParentThreadIDHeader))
				require.Equal(t, "collab_spawn", headers.Get(openAISubagentHeader))
				require.Equal(t, want, gjson.Get(headers.Get(openAIWSTurnMetadataHeader), "parent_thread_id").String())
			})
		}
	}
}

// 父子关系同账号同用户才匹配；切号和换用户不会查到原来的随机收敛关系。
func TestCodexMetadataRepairLineageIsolation(t *testing.T) {
	for _, mode := range []string{"off", "device", "session", "full"} {
		t.Run(mode, func(t *testing.T) {
			c, account := metadataRepairTestContext()
			account.Extra[codexFingerprintModeExtraKey] = mode
			account.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
			c.Set("api_key", &APIKey{ID: 401})
			prefix := t.Name()
			parentBody, _ := json.Marshal(map[string]any{"client_metadata": map[string]any{"session_id": "session-" + prefix, "thread_id": "parent-" + prefix, "turn_id": "turn-" + prefix}})
			parent := buildCodexMetadataRepair(c, account, parentBody, "", true)
			require.NotNil(t, parent)
			_, err := parent.applyRaw(parentBody)
			require.NoError(t, err)
			childMetadata, _ := json.Marshal(map[string]any{"thread_id": "child-" + prefix, "parent_thread_id": "parent-" + prefix, "forked_from_thread_id": "parent-" + prefix, "parent_turn_id": "turn-" + prefix, "root_turn_id": "turn-" + prefix, "subagent_kind": "collab_spawn"})
			childBody, _ := json.Marshal(map[string]any{"client_metadata": map[string]any{openAIWSTurnMetadataHeader: string(childMetadata)}})
			child := buildCodexMetadataRepair(c, account, childBody, "", true)
			require.NotNil(t, child)
			nested := gjson.Parse(child.metadata[openAIWSTurnMetadataHeader].(string))
			require.Equal(t, parent.metadata["thread_id"], nested.Get("parent_thread_id").String())
			require.Equal(t, parent.metadata["thread_id"], nested.Get("forked_from_thread_id").String())
			require.Equal(t, parent.metadata["turn_id"], nested.Get("parent_turn_id").String())
			require.Equal(t, parent.metadata["turn_id"], nested.Get("root_turn_id").String())
			for _, boundary := range []string{"tenant", "account"} {
				other := *account
				if boundary == "tenant" {
					c.Set("api_key", &APIKey{ID: 402})
				} else {
					c.Set("api_key", &APIKey{ID: 401})
					other.Credentials = map[string]any{"chatgpt_account_id": "different-account", "access_token": "synthetic"}
				}
				next := buildCodexMetadataRepair(c, &other, childBody, "", true)
				require.NotNil(t, next)
				value := gjson.Get(next.metadata[openAIWSTurnMetadataHeader].(string), "parent_thread_id").String()
				require.NotEqual(t, parent.metadata["thread_id"], value, boundary)
				require.Equal(t, scopeCodexAccountIdentityValue(&other, getAPIKeyIDFromContext(c), "thread", "parent-"+prefix), value)
			}
		})
	}
}

// Memory 嵌套对象只保留官方允许的显式回合，不从 flat/握手补造身份。
func TestCodexMetadataRepairMemoryShape(t *testing.T) {
	for _, mode := range []string{"off", "device", "session", "full"} {
		for _, explicitTurn := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/turn=%v", mode, explicitTurn), func(t *testing.T) {
				c, account := metadataRepairTestContext()
				account.Extra[codexFingerprintModeExtraKey] = mode
				account.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
				nested := map[string]any{"request_kind": "memory", "extension": "keep", "turn_started_at_unix_ms": 1234}
				if explicitTurn {
					nested["turn_id"] = "explicit-turn"
				}
				raw, _ := json.Marshal(nested)
				body, _ := json.Marshal(map[string]any{"input": "keep", "client_metadata": map[string]any{"session_id": "session", "thread_id": "thread", "x-codex-installation-id": "device", "x-codex-window-id": "window", openAIWSTurnMetadataHeader: string(raw)}})
				prepareCodexMetadataRepair(c, account, body, "")
				svc := &OpenAIGatewayService{}
				req, err := svc.buildUpstreamRequest(context.Background(), c, account, body, "synthetic-token", true, "", true)
				require.NoError(t, err)
				out, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				req.Body.Close()
				for _, value := range []string{gjson.GetBytes(out, "client_metadata.x-codex-turn-metadata").String(), req.Header.Get(openAIWSTurnMetadataHeader)} {
					for _, field := range []string{"installation_id", "session_id", "thread_id", "window_id", "agent_name", "window_number", "context_window_id"} {
						require.False(t, gjson.Get(value, field).Exists(), field)
					}
					require.Equal(t, explicitTurn, gjson.Get(value, "turn_id").Exists())
					require.Equal(t, int64(1234), gjson.Get(value, "turn_started_at_unix_ms").Int())
					require.Equal(t, "keep", gjson.Get(value, "extension").String())
				}
				require.Equal(t, "keep", gjson.GetBytes(out, "input").String())
			})
		}
	}
}

// 当前 body 的父子关系优先，后续帧缺失时不能用旧握手关系覆盖新轮。
func TestCodexMetadataRepairLineageSourcesAndRetry(t *testing.T) {
	c, a := metadataRepairTestContext()
	c.Request.Header.Set(codexParentThreadIDHeader, "header-parent")
	c.Request.Header.Set(openAISubagentHeader, "review")
	c.Request.Header.Set(openAIWSTurnMetadataHeader, `{"parent_thread_id":"embedded-header-parent","subagent_kind":"review"}`)
	body := []byte(`{"client_metadata":{"x-codex-parent-thread-id":"flat-parent","x-openai-subagent":"flat-kind","parent_thread_id":"flat-parent","x-codex-turn-metadata":"{\"parent_thread_id\":\"body-parent\",\"subagent_kind\":\"collab_spawn\"}"}}`)
	prepareCodexMetadataRepair(c, a, body, "")
	snapshot := stagedCodexMetadataRepair(c, a)
	require.NotNil(t, snapshot)
	want := scopeCodexAccountIdentityValue(a, 0, "thread", "body-parent")
	require.Equal(t, want, snapshot.headers.Get(codexParentThreadIDHeader))
	require.Equal(t, want, snapshot.metadata["parent_thread_id"])
	require.Equal(t, "flat-kind", snapshot.headers.Get(openAISubagentHeader), "兼容头和内层 kind 独立保留")
	prepareCodexMetadataRepair(c, a, body, "")
	require.Same(t, snapshot, stagedCodexMetadataRepair(c, a))
	c.Request.Header.Set(codexParentThreadIDHeader, "changed-header")
	prepareCodexMetadataRepair(c, a, body, "")
	require.NotSame(t, snapshot, stagedCodexMetadataRepair(c, a))
	next := buildCodexMetadataRepair(c, a, []byte(`{"client_metadata":{"turn_id":"next-turn"}}`), "", false)
	require.NotNil(t, next)
	headers := snapshot.headers.Clone()
	next.applyHeaders(headers)
	require.Empty(t, headers.Get(codexParentThreadIDHeader))
	require.Empty(t, headers.Get(openAISubagentHeader))
	require.NotContains(t, next.metadata, "parent_thread_id")
	c.Request.Header.Del(openAIWSTurnMetadataHeader)
	for _, invalid := range []string{"bad\r\nvalue", "bad\x01value", strings.Repeat("x", codexMetadataRepairMaxBytes+1)} {
		c.Request.Header.Set(codexParentThreadIDHeader, invalid)
		snapshot := buildCodexMetadataRepair(c, a, []byte(`{}`), "", true)
		require.NotNil(t, snapshot, "非法可选父引用不能关闭其它修复")
		require.Empty(t, snapshot.headers.Get(codexParentThreadIDHeader))
	}
}

// 关闭时不仅不投影兼容头，也不修改主 JSON 中原有的 Memory 形态。
func TestCodexMetadataRepairMemoryDisabledAndNoSyntheticTurn(t *testing.T) {
	c, a := metadataRepairTestContext()
	body := []byte(`{"client_metadata":{"session_id":"flat-session","thread_id":"flat-thread","x-codex-turn-metadata":"{\"request_kind\":\"memory\"}"}}`)
	c.Request.Header.Set(openAIWSTurnMetadataHeader, `{"turn_id":"stale-handshake","turn_started_at_unix_ms":1}`)
	snapshot := buildCodexMetadataRepair(c, a, body, "", true)
	require.NotNil(t, snapshot)
	require.NotContains(t, snapshot.metadata, "turn_id")
	nested := gjson.Parse(snapshot.metadata[openAIWSTurnMetadataHeader].(string))
	require.False(t, nested.Get("turn_id").Exists())
	require.False(t, nested.Get("turn_started_at_unix_ms").Exists())
	a.Extra[codexMetadataRepairExtraKey] = false
	prepareCodexMetadataRepair(c, a, body, "")
	out, err := stagedCodexMetadataRepair(c, a).applyRaw(body)
	require.NoError(t, err)
	require.Equal(t, string(body), string(out))
}

// 每轮关系不再作为固定握手身份；关闭仍使用原连接池判定。
func TestCodexMetadataRepairLineagePoolCompatibility(t *testing.T) {
	_, a := metadataRepairTestContext()
	headers := make(http.Header)
	base := normalizeOpenAIWSHandshakeCompatibility(a, headers)
	for _, field := range []string{codexParentThreadIDHeader, openAISubagentHeader} {
		changed := headers.Clone()
		changed.Set(field, "different")
		require.Equal(t, base, normalizeOpenAIWSHandshakeCompatibility(a, changed))
		a.Extra[codexMetadataRepairExtraKey] = false
		require.Equal(t, normalizeOpenAIWSHandshakeCompatibility(a, headers), normalizeOpenAIWSHandshakeCompatibility(a, changed))
		a.Extra[codexMetadataRepairExtraKey] = true
	}
}

// 相同凭据的影子共享映射，AT 与其它租户仍受 namespace 限制；并发读取安全。
func TestCodexMetadataRepairLineageShadowATAndConcurrent(t *testing.T) {
	for _, accountType := range []string{AccountTypeOAuth, AccountTypeSetupToken} {
		c, a := metadataRepairTestContext()
		a.Type = accountType
		a.Extra[codexFingerprintModeExtraKey] = "session"
		a.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
		c.Set("api_key", &APIKey{ID: 811})
		raw := t.Name() + accountType
		body, _ := json.Marshal(map[string]any{"client_metadata": map[string]any{"thread_id": raw, "turn_id": raw}})
		parent := buildCodexMetadataRepair(c, a, body, "session-parent", true)
		require.NotNil(t, parent)
		out, err := parent.applyRaw(body)
		require.NoError(t, err)
		require.NotEmpty(t, out)
		childBody, _ := json.Marshal(map[string]any{"client_metadata": map[string]any{codexParentThreadIDHeader: raw, "parent_turn_id": raw}})
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				next := buildCodexMetadataRepair(c, a, childBody, "", true)
				if next == nil || next.headers.Get(codexParentThreadIDHeader) != parent.metadata["thread_id"] {
					t.Error("并发父线程映射不一致")
				}
			}()
		}
		wg.Wait()
		if accountType == AccountTypeOAuth {
			shadow := *a
			shadow.ID = 991
			shadow.ParentAccountID = &a.ID
			shadow.Credentials = nil
			c.Set(codexAccountIdentitySourceContextKey, a)
			next := buildCodexMetadataRepair(c, &shadow, childBody, "", true)
			require.NotNil(t, next)
			require.Equal(t, parent.metadata["thread_id"], next.headers.Get(codexParentThreadIDHeader))
		}
	}
}
