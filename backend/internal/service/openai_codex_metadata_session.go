package service

import (
	"crypto/sha256"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// 稳定关系属于线程；父/根回合、请求类型、时间和环境快照不得跨轮继承。
var codexMetadataStableLineageFields = []string{"parent_thread_id", "forked_from_thread_id", "subagent_kind", openAISubagentHeader}

func cloneCodexMetadataMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// null/空串表示客户端显式清除；非字符串不是缺失，不能用旧身份掩盖它。
func codexMetadataIdentityValue(cm, nested map[string]any, field string, aliases []string) (string, bool, bool) {
	candidates := []map[string]any{nested}
	names := []string{field}
	for _, alias := range aliases {
		candidates = append(candidates, cm)
		names = append(names, alias)
	}
	for i, candidate := range candidates {
		value, exists := candidate[names[i]]
		if !exists {
			continue
		}
		if value == nil {
			return "", true, true
		}
		if _, ok := value.(string); !ok {
			return "", true, false
		}
		return codexMetadataString(value), true, true
	}
	return "", false, true
}

func codexMetadataFieldSupplied(cm, nested map[string]any, field string) bool {
	if _, ok := nested[field]; ok {
		return true
	}
	for _, entry := range codexMetadataLineageFields {
		if entry.field == field {
			for _, alias := range entry.aliases {
				if _, ok := cm[alias]; ok {
					return true
				}
			}
		}
	}
	return false
}

// 一个实例只供单条 WS 入站连接的上行帧串行使用；不写共享 Gin，不跨连接存储。
type codexMetadataSession struct {
	owner  [sha256.Size]byte
	stable map[string]any
}

const codexMetadataReplayContextKey = "codex_metadata_replay_defaults"

// 仅当前入站 WS 的一次安全重放携带原始默认值；绝不写进原始 payload，关闭目标账号不读取。
type codexMetadataReplay struct {
	ingress     *gin.Context
	apiKeyID    int64
	payloadHash [sha256.Size]byte
	stable      map[string]any
}

func stageCodexMetadataReplay(c *gin.Context, payload []byte, stable map[string]any) {
	if c == nil || len(payload) == 0 {
		return
	}
	c.Set(codexMetadataReplayContextKey, &codexMetadataReplay{ingress: c, apiKeyID: getAPIKeyIDFromContext(c), payloadHash: sha256.Sum256(payload), stable: cloneCodexMetadataMap(stable)})
}

func newCodexMetadataSession(c *gin.Context, a *Account, payload []byte) *codexMetadataSession {
	state := &codexMetadataSession{}
	if c == nil {
		return state
	}
	value, _ := c.Get(codexMetadataReplayContextKey)
	replay, _ := value.(*codexMetadataReplay)
	c.Set(codexMetadataReplayContextKey, (*codexMetadataReplay)(nil))
	// 旧链路会 Clone/WithContext 替换 http.Request，Gin 入站作用域及一次性消费才是生命周期边界。
	if a.IsCodexMetadataRepairEnabled() && replay != nil && replay.ingress == c &&
		replay.apiKeyID == getAPIKeyIDFromContext(c) && replay.payloadHash == sha256.Sum256(payload) {
		state.owner = codexMetadataRepairSourceKey(c, a, nil, "")
		state.stable = cloneCodexMetadataMap(replay.stable)
	}
	return state
}

func (state *codexMetadataSession) build(c *gin.Context, a *Account, body []byte, first bool) *codexMetadataRepairSnapshot {
	if c == nil || !a.IsCodexMetadataRepairEnabled() || isOpenAIResponsesCompactPath(c) {
		state.stable = nil
		return nil
	}
	// 空 body 的来源键仍包含凭据、API Key、指纹、真实头；换号/换租户不能继承原连接身份。
	owner := codexMetadataRepairSourceKey(c, a, nil, "")
	if state.owner != owner {
		state.owner = owner
		state.stable = nil
	}
	defaults := state.stable
	cm := codexMetadataObject(gjson.GetBytes(body, "client_metadata").Raw)
	nested := codexMetadataObject(codexMetadataString(cm[openAIWSTurnMetadataHeader]))
	kind, kindSupplied, _ := codexMetadataIdentityValue(cm, nested, "subagent_kind", []string{"subagent_kind"})
	if defaults != nil && codexMetadataFieldSupplied(cm, nested, "subagent_kind") && !codexMetadataFieldSupplied(cm, nested, openAISubagentHeader) &&
		kindSupplied && kind != codexMetadataString(defaults["subagent_kind"]) {
		defaults = cloneCodexMetadataMap(defaults)
		delete(defaults, openAISubagentHeader) // 当前明确的新 kind 不得继承旧兼容标记。
	}
	for _, field := range []string{"session_id", "thread_id"} {
		aliases := []string{field}
		if field == "session_id" {
			aliases = append(aliases, "session-id")
		} else {
			aliases = append(aliases, "thread-id")
		}
		value, supplied, valid := codexMetadataIdentityValue(cm, nested, field, aliases)
		if supplied && (!valid || value != codexMetadataString(defaults[field])) && defaults != nil {
			// 明确切线程时保留安装来源；切会话同时清理所有线程来源，且不回落旧握手。
			next := make(map[string]any)
			if v, ok := defaults["installation_id"]; ok {
				next["installation_id"] = v
			}
			if field == "thread_id" {
				if v, ok := defaults["session_id"]; ok {
					next["session_id"] = v
				}
			}
			defaults = next
		}
	}
	update := gjson.GetBytes(body, "type").String() == "session.update"
	if update && cm == nil {
		return nil
	} // 既有兼容更新不凭空加入 metadata。
	snapshot := buildCodexMetadataRepairWithDefaults(c, a, body, "", first, defaults)
	if snapshot != nil {
		// Memory 无稳定线程身份；不能用省略字段清空已有会话，也不能把 Memory 回合写为连接身份。
		kind := gjson.Get(snapshot.metadata[openAIWSTurnMetadataHeader].(string), "request_kind").String()
		if kind != "memory" {
			state.stable = cloneCodexMetadataMap(snapshot.stable)
		}
		if update {
			// session.update 不补造 turn/request_kind/time，完整对象只在原来提供时写回。
			if _, ok := cm[openAIWSTurnMetadataHeader]; !ok {
				delete(snapshot.metadata, openAIWSTurnMetadataHeader)
			}
			// 兼容更新只补其声明的字段，不用完整回合补齐改变更新对象的作用范围。
			for key := range snapshot.metadata {
				if _, exists := cm[key]; !exists {
					delete(snapshot.metadata, key)
				}
			}
		}
	}
	return snapshot
}
