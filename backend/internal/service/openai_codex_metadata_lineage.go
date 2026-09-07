package service

import (
	"net/http"

	"golang.org/x/net/http/httpguts"
)

// 关联只描述元数据，不参与认证、账号选择、prompt_cache_key 或 response 续接。
var codexMetadataLineageFields = []struct {
	field   string
	kind    string
	aliases []string
}{
	{"parent_thread_id", "thread", []string{codexParentThreadIDHeader, "parent_thread_id"}},
	{"forked_from_thread_id", "thread", []string{"forked_from_thread_id"}},
	{"parent_turn_id", "turn", []string{"parent_turn_id"}},
	{"root_turn_id", "turn", []string{"root_turn_id"}},
	{"subagent_kind", "", []string{"subagent_kind"}},
	// 官方兼容头 collab_spawn 和内层 thread_spawn 并非同值别名。
	{openAISubagentHeader, "", []string{openAISubagentHeader}},
}

// 逐字段选当前可信来源；非法高优先级值只省略该字段，不借旧来源覆盖当前意图。
func collectCodexMetadataLineage(cm, bodyTurn, headerTurn map[string]any, headers http.Header, firstTurn bool) map[string]string {
	values := make(map[string]string)
	for _, field := range codexMetadataLineageFields {
		var candidates []any
		if field.field != openAISubagentHeader {
			candidates = append(candidates, bodyTurn[field.field])
		}
		for _, alias := range field.aliases {
			candidates = append(candidates, cm[alias])
		}
		if firstTurn {
			if field.field != openAISubagentHeader {
				candidates = append(candidates, headerTurn[field.field])
			}
			for _, alias := range field.aliases {
				candidates = append(candidates, headers.Get(alias))
			}
		}
		for _, candidate := range candidates {
			if candidate == nil {
				continue
			}
			value, ok := candidate.(string)
			if !ok {
				break
			}
			value = codexMetadataString(value)
			if value == "" {
				continue
			}
			if !httpguts.ValidHeaderFieldValue(value) || len(value) > codexMetadataRepairMaxBytes {
				break
			}
			values[field.field] = value
			break
		}
	}
	// 只推导官方明确的对应关系；未知扩展分别保留，不猜测客户端角色。
	if values[openAISubagentHeader] == "" {
		switch kind := values["subagent_kind"]; kind {
		case "thread_spawn":
			values[openAISubagentHeader] = "collab_spawn"
		case "review", "compact", "memory_consolidation":
			values[openAISubagentHeader] = kind
		}
	}
	// 独立头还可表示 Internal 来源（如 guardian/memory），不能据此补造内层 kind。
	return values
}

// 主编号和引用使用相同的无状态账号/API Key 隔离。
// 不记录“已发送”状态：重试、并发、重启、多实例与长会话不会依赖缓存命中。
func projectCodexMetadataLineage(cm, turn map[string]any, headers http.Header, raw map[string]string, account *Account, apiKeyID int64) {
	headers.Set(codexParentThreadIDHeader, "")
	headers.Set(openAISubagentHeader, "")
	for _, field := range codexMetadataLineageFields {
		existingAliases := make(map[string]bool, len(field.aliases))
		delete(turn, field.field)
		for _, alias := range field.aliases {
			_, existingAliases[alias] = cm[alias]
			delete(cm, alias)
		}
		value := raw[field.field]
		if value == "" {
			continue
		}
		if field.kind != "" {
			value = scopeCodexAccountIdentityValue(account, apiKeyID, field.kind, value)
		}
		if field.field != openAISubagentHeader {
			turn[field.field] = value
		}
		// 官方 flat 没有 forked_from_thread_id 或 subagent_kind，不凭空增加。
		if field.field != "forked_from_thread_id" && field.field != "subagent_kind" {
			cm[field.aliases[0]] = value
		}
		for _, alias := range field.aliases {
			if existingAliases[alias] {
				cm[alias] = value
			}
		}
		if field.field == "parent_thread_id" || field.field == openAISubagentHeader {
			headers.Set(field.aliases[0], value)
		}
	}
}
