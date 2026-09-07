package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 用官方字段组合验证，避免用错误夹具掩盖兼容头和内层 kind 的差别。
func TestCodexMetadataRepairOfficialSubagentKinds(t *testing.T) {
	for _, tc := range []struct{ kind, header, wantKind, wantHeader string }{
		{"thread_spawn", "collab_spawn", "thread_spawn", "collab_spawn"},
		{"thread_spawn", "", "thread_spawn", "collab_spawn"},
		{"", "collab_spawn", "", "collab_spawn"},
		{"", "guardian", "", "guardian"},
		{"", "memory_consolidation", "", "memory_consolidation"},
		{"custom-kind", "custom-header", "custom-kind", "custom-header"},
	} {
		t.Run(tc.kind+"/"+tc.header, func(t *testing.T) {
			c, a := metadataRepairTestContext()
			cm := map[string]any{}
			turn := map[string]any{"request_kind": "turn"}
			if tc.kind != "" {
				turn["subagent_kind"] = tc.kind
			}
			if tc.header != "" {
				cm[openAISubagentHeader] = tc.header
			}
			blob, _ := json.Marshal(turn)
			cm[openAIWSTurnMetadataHeader] = string(blob)
			body, _ := json.Marshal(map[string]any{"client_metadata": cm})
			s := buildCodexMetadataRepair(c, a, body, "", true)
			require.NotNil(t, s)
			require.Equal(t, tc.wantHeader, s.headers.Get(openAISubagentHeader))
			require.Equal(t, tc.wantKind, gjson.Get(s.metadata[openAIWSTurnMetadataHeader].(string), "subagent_kind").String())
		})
	}
}

// 子先父后、未发送重建与 Memory self-root 均无需进程登记；不同用户/账号仍隔离。
func TestCodexMetadataRepairStatelessLineage(t *testing.T) {
	for _, mode := range []string{"off", "device", "session", "full"} {
		for _, kind := range []string{"turn", "memory"} {
			t.Run(mode+"/"+kind, func(t *testing.T) {
				c, a := metadataRepairTestContext()
				a.Extra[codexFingerprintModeExtraKey] = mode
				a.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
				c.Set("api_key", &APIKey{ID: 711})
				nested := map[string]any{"thread_id": "parent-thread", "turn_id": "parent-turn", "root_turn_id": "parent-turn", "request_kind": kind}
				blob, _ := json.Marshal(nested)
				parentBody, _ := json.Marshal(map[string]any{"client_metadata": map[string]any{openAIWSTurnMetadataHeader: string(blob)}})
				childBody := []byte(`{"client_metadata":{"thread_id":"child-thread","parent_turn_id":"parent-turn","x-codex-parent-thread-id":"parent-thread"}}`)
				child := buildCodexMetadataRepair(c, a, childBody, "", true)
				parent := buildCodexMetadataRepair(c, a, parentBody, "", true)
				require.NotNil(t, child)
				require.NotNil(t, parent)
				p := gjson.Parse(parent.metadata[openAIWSTurnMetadataHeader].(string))
				require.Equal(t, p.Get("turn_id").String(), p.Get("root_turn_id").String())
				require.Equal(t, parent.metadata["turn_id"], child.metadata["parent_turn_id"])
				if kind == "turn" {
					require.Equal(t, parent.metadata["thread_id"], child.metadata[codexParentThreadIDHeader])
				}
				// 不执行 applyRaw/发送，用全新上下文模拟另一实例；编号仍相同。
				c2, _ := metadataRepairTestContext()
				c2.Set("api_key", &APIKey{ID: 711})
				again := buildCodexMetadataRepair(c2, a, parentBody, "", true)
				require.Equal(t, parent.metadata["turn_id"], again.metadata["turn_id"])
				c2.Set("api_key", &APIKey{ID: 712})
				other := buildCodexMetadataRepair(c2, a, parentBody, "", true)
				require.NotEqual(t, parent.metadata["turn_id"], other.metadata["turn_id"])
				if kind == "turn" {
					require.NotEqual(t, parent.metadata["thread_id"], other.metadata["thread_id"])
				}
			})
		}
	}
}

// 单个可选关系不合法时其它合法关系继续修复，不能由一个坏字段撤销整个开关。
func TestCodexMetadataRepairOptionalFieldIsolation(t *testing.T) {
	c, a := metadataRepairTestContext()
	c.Request.Header.Set(openAIWSTurnMetadataHeader, "bad-json")
	body := []byte(`{"client_metadata":{"x-codex-turn-metadata":"{\"turn_id\":\"turn\",\"parent_turn_id\":7,\"root_turn_id\":\"turn\",\"subagent_kind\":\"thread_spawn\"}"}}`)
	s := buildCodexMetadataRepair(c, a, body, "", true)
	require.NotNil(t, s)
	turn := gjson.Parse(s.metadata[openAIWSTurnMetadataHeader].(string))
	require.False(t, turn.Get("parent_turn_id").Exists())
	require.Equal(t, turn.Get("turn_id").String(), turn.Get("root_turn_id").String())
	require.Equal(t, "collab_spawn", s.headers.Get(openAISubagentHeader))
}

// 收敛模式即使没有客户端身份，也不能让不同用户共用一个未知线程连接。
func TestCodexMetadataRepairMissingThreadPoolIsolation(t *testing.T) {
	for _, mode := range []string{"session", "full"} {
		var previous http.Header
		for _, id := range []int64{11, 12} {
			c, a := metadataRepairTestContext()
			a.Extra[codexFingerprintModeExtraKey] = mode
			a.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
			c.Set("api_key", &APIKey{ID: id})
			s := buildCodexMetadataRepair(c, a, []byte(`{}`), "", true)
			require.NotNil(t, s)
			require.NotEmpty(t, s.headers.Get("x-client-request-id"))
			require.Empty(t, s.headers.Get("thread-id"))
			if previous != nil {
				require.NotEqual(t, normalizeOpenAIWSHandshakeCompatibility(a, previous), normalizeOpenAIWSHandshakeCompatibility(a, s.headers))
			}
			previous = s.headers
		}
	}
}

// 请求级父子关系存在、变化或省略，不得使真实线程未变的严格续接失效。
func TestCodexMetadataRepairWSLineageReconnect(t *testing.T) {
	for _, mode := range []string{"off", "device", "session", "full"} {
		t.Run(mode, func(t *testing.T) {
			c, a := metadataRepairTestContext()
			a.Extra[codexFingerprintModeExtraKey] = mode
			a.Extra[codexFingerprintSeedExtraKey] = "11111111-1111-4111-8111-111111111111"
			cfg := newOpenAIWSV2TestConfig()
			cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 1
			cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
			cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 1
			pool := newOpenAIWSConnPool(cfg)
			defer pool.Close()
			dialer := &openAIWSCountingDialer{}
			pool.setClientDialerForTest(dialer)
			svc := &OpenAIGatewayService{cfg: cfg}
			var connID string
			for i, parent := range []string{"parent-one", "", "parent-two"} {
				nested, _ := json.Marshal(map[string]any{"session_id": "session", "thread_id": "thread", "turn_id": fmt.Sprint(i), "parent_thread_id": parent, "subagent_kind": "thread_spawn"})
				body, _ := json.Marshal(map[string]any{"client_metadata": map[string]any{openAIWSTurnMetadataHeader: string(nested)}})
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
				prepareCodexMetadataRepair(c, a, body, "")
				h, _, err := svc.buildOpenAIWSHeaders(context.Background(), c, a, "synthetic", OpenAIWSProtocolDecision{}, true, "", "", "", "gpt-6-astra", "")
				require.NoError(t, err)
				for _, key := range []string{openAIWSTurnMetadataHeader, codexParentThreadIDHeader, openAISubagentHeader} {
					require.Empty(t, h.Get(key))
				}
				lease, err := pool.Acquire(context.Background(), openAIWSAcquireRequest{Account: a, WSURL: "wss://synthetic.invalid/responses", Headers: h, PreferredConnID: connID, ForcePreferredConn: i > 0})
				require.NoError(t, err)
				if i == 0 {
					connID = lease.ConnID()
				} else {
					require.Equal(t, connID, lease.ConnID())
				}
				lease.Release()
			}
			require.Equal(t, 1, dialer.DialCount())
		})
	}
}
