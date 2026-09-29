package postgres

import (
	"encoding/json"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"maps"
	"strconv"
	"strings"
)

// 原有写入配置策略迁入同一所有者，数值/前缀/SQL表达式保持不变。
var schedulerNeutralExtraKeyPrefixes = []string{
	"codex_primary_",
	"codex_secondary_",
	"codex_5h_",
	"codex_7d_",
	"codex_reset_credit_",
	"passive_usage_",
	"ollama_cloud_usage",
	"cn_usage_monitor",
}

const (
	// 废弃账号扩展键仅用于写入边界清理，仓储不得再赋予它们业务语义。
	deprecatedUpstreamBillingProbeExtraKey        = "upstream_billing_probe"
	deprecatedUpstreamBillingProbeEnabledExtraKey = "upstream_billing_probe_enabled"
	deprecatedOpenAILongContextBillingExtraKey    = "openai_long_context_billing_enabled"
)

func discardDeprecatedAccountExtra(extra map[string]any) {
	delete(extra, deprecatedUpstreamBillingProbeExtraKey)
	delete(extra, deprecatedUpstreamBillingProbeEnabledExtraKey)
	delete(extra, deprecatedOpenAILongContextBillingExtraKey)
}

var schedulerNeutralExtraKeys = map[string]struct{}{
	"codex_usage_updated_at":                {},
	"grok_billing_snapshot":                 {},
	"qoder_quota_snapshot":                  {},
	"qoder_quota_updated_at":                {},
	"session_window_utilization":            {},
	provider.CNUsageMonitorSnapshotExtraKey: {},
}

const codexFingerprintSeedCanonicalPattern = "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
const codexFingerprintNilSeed = "00000000-0000-0000-0000-000000000000"

func codexFingerprintSeedValidSQL(extraExpr string) string {
	value := "(" + extraExpr + " ->> 'codex_fingerprint_seed')"
	return "(" + value + " ~ '" + codexFingerprintSeedCanonicalPattern + "' AND " + value + " <> '" + codexFingerprintNilSeed + "')"
}

// ensureCodexFingerprintSeedSQL 在同一条 SQL 中保留合法 seed，避免并发更新产生身份漂移。
func ensureCodexFingerprintSeedSQL(extraExpr string) string {
	return "CASE WHEN platform = 'openai' AND type = 'oauth' THEN " +
		"jsonb_set(" + extraExpr + ", '{codex_fingerprint_seed}', " +
		"CASE WHEN " + codexFingerprintSeedValidSQL("extra") +
		" THEN to_jsonb(extra ->> 'codex_fingerprint_seed') ELSE to_jsonb(gen_random_uuid()::text) END, true) " +
		"ELSE " + extraExpr + " END"
}

func stripCodexFingerprintSeedFromExtraUpdate(extra map[string]any) map[string]any {
	if extra == nil {
		return nil
	}
	if _, exists := extra["codex_fingerprint_seed"]; !exists {
		return extra
	}
	stripped := make(map[string]any, len(extra)-1)
	for key, value := range extra {
		if key != "codex_fingerprint_seed" {
			stripped[key] = value
		}
	}
	return stripped
}

func shouldEnqueueSchedulerOutboxForExtraUpdates(updates map[string]any) bool {
	if len(updates) == 0 {
		return false
	}
	for key := range updates {
		if isSchedulerNeutralExtraKey(key) {
			continue
		}
		return true
	}
	return false
}

func isSchedulerNeutralExtraKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	if _, ok := schedulerNeutralExtraKeys[key]; ok {
		return true
	}
	for _, prefix := range schedulerNeutralExtraKeyPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func cnUsageMonitorIdentityExtraPatch(updates map[string]any) bool {
	for _, key := range []string{
		provider.UpstreamUsageQueryExtraKey,
		"enable_tls_fingerprint",
		"tls_fingerprint_profile_id",
		"tls_fingerprint_router_id",
	} {
		if _, ok := updates[key]; ok {
			return true
		}
	}
	return false
}

func ollamaCloudUsageSnapshotClearRequested(extra map[string]any) bool {
	value, ok := extra[provider.OllamaCloudUsageSnapshotExtraKey]
	return ok && value == nil
}

func decodeAccountExtraJSON(raw []byte) (any, bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, false, nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, false, err
	}
	return value, true, nil
}

const ollamaCloudBaseURLRegexSQL = `^[hH][tT][tT][pP][sS]://([wW][wW][wW]\.)?[oO][lL][lL][aA][mM][aA]\.[cC][oO][mM](:443)?(/v1)?$`

func ollamaCloudBaseURLMatchesSQL(expression string) string {
	return "btrim(" + expression + ") ~ '" + ollamaCloudBaseURLRegexSQL + "'"
}
func copyJSONMap(value map[string]any) map[string]any { return maps.Clone(value) }
func itoa(value int) string                           { return strconv.Itoa(value) }
func joinClauses(values []string, sep string) string  { return strings.Join(values, sep) }
func buildSchedulerGroupPayload(ids []int64) any {
	if len(ids) == 0 {
		return nil
	}
	return map[string]any{"group_ids": ids}
}
