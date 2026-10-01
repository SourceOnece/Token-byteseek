package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// legacyConfigKeys 只映射已有明确替代项的部署配置，不匹配用户自定义名称。
var legacyConfigKeys = []struct {
	oldKey string
	newKey string
}{
	{"pricing.hash_check_interval_minutes", "pricing.check_interval_minutes"},
	{"gateway.max_account_switches", "gateway.max_provider_switches"},
	{"gateway.max_account_switches_gemini", "gateway.max_provider_switches_gemini"},
	{"gateway.openai_ws.max_conns_per_account", "gateway.openai_ws.max_conns_per_provider"},
	{"gateway.openai_ws.min_idle_per_account", "gateway.openai_ws.min_idle_per_provider"},
	{"gateway.openai_ws.max_idle_per_account", "gateway.openai_ws.max_idle_per_provider"},
	{"gateway.openai_ws.dynamic_max_conns_by_account_concurrency_enabled", "gateway.openai_ws.dynamic_max_conns_by_provider_concurrency_enabled"},
	{"gateway.openai_ws.lb_top_k", "gateway.advanced_scheduler.lb_top_k"},
	{"gateway.openai_ws.scheduler_score_weights.priority", "gateway.advanced_scheduler.score_weights.priority"},
	{"gateway.openai_ws.scheduler_score_weights.load", "gateway.advanced_scheduler.score_weights.load"},
	{"gateway.openai_ws.scheduler_score_weights.queue", "gateway.advanced_scheduler.score_weights.queue"},
	{"gateway.openai_ws.scheduler_score_weights.error_rate", "gateway.advanced_scheduler.score_weights.error_rate"},
	{"gateway.openai_ws.scheduler_score_weights.ttft", "gateway.advanced_scheduler.score_weights.ttft"},
	{"gateway.openai_ws.scheduler_score_weights.reset", "gateway.advanced_scheduler.score_weights.reset"},
	{"gateway.openai_ws.scheduler_score_weights.quota_headroom", "gateway.advanced_scheduler.score_weights.quota_headroom"},
	{"gateway.openai_ws.scheduler_score_weights.previous_response", "gateway.advanced_scheduler.score_weights.previous_response"},
	{"gateway.openai_ws.scheduler_score_weights.session_sticky", "gateway.advanced_scheduler.score_weights.session_sticky"},
	{"gateway.openai_scheduler.sticky_escape_enabled", "gateway.advanced_scheduler.sticky_escape_enabled"},
	{"gateway.openai_scheduler.sticky_escape_ttft_ms", "gateway.advanced_scheduler.sticky_escape_ttft_ms"},
	{"gateway.openai_scheduler.sticky_escape_error_rate", "gateway.advanced_scheduler.sticky_escape_error_rate"},
}

// applyLegacyConfigCompatibility 在加载边界统一旧名称，不改写部署文件或环境变量。
// 环境变量优先于 YAML；同一来源的新名称优先。使用字段是否存在判断，保留零值和 false。
// @project-doc docs/interfaces/configuration.md#configuration_sources
func applyLegacyConfigCompatibility() error {
	for _, item := range legacyConfigKeys {
		newEnv := strings.ToUpper(strings.ReplaceAll(item.newKey, ".", "_"))
		oldEnv := strings.ToUpper(strings.ReplaceAll(item.oldKey, ".", "_"))
		if err := viper.BindEnv(item.newKey, newEnv, oldEnv); err != nil {
			return fmt.Errorf("bind configuration aliases for %s: %w", item.newKey, err)
		}
		if viper.InConfig(item.newKey) || !viper.InConfig(item.oldKey) {
			continue
		}
		// 合并到内存中的 YAML 层，避免 Set 的最高优先级遮蔽环境变量。
		parts := strings.Split(item.newKey, ".")
		patch := map[string]any{}
		parent := patch
		for _, part := range parts[:len(parts)-1] {
			child := map[string]any{}
			parent[part] = child
			parent = child
		}
		parent[parts[len(parts)-1]] = viper.Get(item.oldKey)
		if err := viper.MergeConfigMap(patch); err != nil {
			return fmt.Errorf("map configuration %s to %s: %w", item.oldKey, item.newKey, err)
		}
	}
	return nil
}

// normalizeLegacyConnectionPoolIsolation 保持旧部署的连接池隔离粒度。
func normalizeLegacyConnectionPoolIsolation(value string) string {
	switch value {
	case "account":
		return ConnectionPoolIsolationProvider
	case "account_proxy":
		return ConnectionPoolIsolationProviderProxy
	default:
		return value
	}
}
