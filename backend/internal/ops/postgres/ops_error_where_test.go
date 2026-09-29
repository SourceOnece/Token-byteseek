package postgres

import (
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/ops"
)

func TestBuildOpsErrorLogsWhere_UserScopedFilters(t *testing.T) {
	uid := int64(42)
	kid := int64(7)
	filter := &ops.OpsErrorLogFilter{
		UserID:             &uid,
		APIKeyID:           &kid,
		Model:              "claude-sonnet-4-5",
		ExcludeCountTokens: true,
		ErrorPhasesAny:     []string{"auth"},
		ErrorTypesAny:      []string{"rate_limit_error"},
		View:               "all",
	}
	where, args := buildOpsErrorLogsWhere(filter)

	for _, want := range []string{
		"e.user_id = $",
		"e.api_key_id = $",
		"COALESCE(e.requested_model, e.model, '') = $",
		"COALESCE(e.is_count_tokens, false) = false",
		"e.error_phase = ANY($",
		"e.error_type = ANY($",
	} {
		if !strings.Contains(where, want) {
			t.Fatalf("where missing %q\nfull: %s", want, where)
		}
	}
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d", len(args))
	}
}

func TestBuildOpsErrorLogsWhere_ModelFuzzy(t *testing.T) {
	// 默认（ModelFuzzy=false）保持精确匹配
	exact := &ops.OpsErrorLogFilter{Model: "claude"}
	whereExact, _ := buildOpsErrorLogsWhere(exact)
	if !strings.Contains(whereExact, "COALESCE(e.requested_model, e.model, '') = $") {
		t.Fatalf("default should be exact match, got: %s", whereExact)
	}

	// ModelFuzzy=true → ILIKE
	fuzzy := &ops.OpsErrorLogFilter{Model: "claude", ModelFuzzy: true}
	whereFuzzy, args := buildOpsErrorLogsWhere(fuzzy)
	if !strings.Contains(whereFuzzy, "COALESCE(e.requested_model, e.model, '') ILIKE $") {
		t.Fatalf("ModelFuzzy should use ILIKE, got: %s", whereFuzzy)
	}
	if len(args) != 1 || args[0] != "%claude%" {
		t.Fatalf("expected arg \"%%claude%%\", got %v", args)
	}

	// 通配符转义：输入含 % 应被转义为字面量
	esc := &ops.OpsErrorLogFilter{Model: "50%off", ModelFuzzy: true}
	_, escArgs := buildOpsErrorLogsWhere(esc)
	if len(escArgs) != 1 || escArgs[0] != `%50\%off%` {
		t.Fatalf("expected escaped arg, got %v", escArgs)
	}

	esc2 := &ops.OpsErrorLogFilter{Model: "gpt_4o", ModelFuzzy: true}
	_, escArgs2 := buildOpsErrorLogsWhere(esc2)
	if len(escArgs2) != 1 || escArgs2[0] != `%gpt\_4o%` {
		t.Fatalf("underscore should be escaped, got %v", escArgs2)
	}
}

// TestBuildOpsErrorLogsWhere_CyberPolicyStatusExemption 验证流式 cyber_policy
// 命中即使状态码为 200，也不会被客户端错误守卫从管理端和用户端列表排除。
func TestBuildOpsErrorLogsWhere_CyberPolicyStatusExemption(t *testing.T) {
	// 默认过滤必须保留 cyber_policy 豁免和其它错误的状态码守卫。
	where, _ := buildOpsErrorLogsWhere(&ops.OpsErrorLogFilter{})
	if !strings.Contains(where, "e.error_type = 'cyber_policy'") {
		t.Fatalf("default filter must exempt cyber_policy from status >= 400 guard\nfull: %s", where)
	}
	if !strings.Contains(where, "COALESCE(e.status_code, 0) >= 400") {
		t.Fatalf("default filter must still include the status >= 400 guard for non-cyber rows\nfull: %s", where)
	}

	// 未显式允许 recovered upstream 时，phase=upstream 仍保留状态码守卫。
	whereUpstream, _ := buildOpsErrorLogsWhere(&ops.OpsErrorLogFilter{Phase: "upstream"})
	if !strings.Contains(whereUpstream, "COALESCE(e.status_code, 0) >= 400") {
		t.Fatalf("upstream phase without IncludeRecoveredUpstream must keep the status guard\nfull: %s", whereUpstream)
	}
	if !strings.Contains(whereUpstream, "e.error_phase = $") {
		t.Fatalf("upstream phase filter must emit the error_phase condition\nfull: %s", whereUpstream)
	}

	// Ops 专用上游列表显式允许 recovered upstream 后才跳过状态码守卫。
	whereRecovered, _ := buildOpsErrorLogsWhere(&ops.OpsErrorLogFilter{Phase: "upstream", IncludeRecoveredUpstream: true})
	if strings.Contains(whereRecovered, "COALESCE(e.status_code, 0) >= 400") {
		t.Fatalf("upstream phase with IncludeRecoveredUpstream must not add the client-visible status guard\nfull: %s", whereRecovered)
	}

	// provider_auth 使用相同的提供方健康显式开关，但仍与推理上游错误保持独立阶段。
	whereProviderAuth, _ := buildOpsErrorLogsWhere(&ops.OpsErrorLogFilter{Phase: "provider_auth", IncludeRecoveredUpstream: true})
	if strings.Contains(whereProviderAuth, "COALESCE(e.status_code, 0) >= 400") {
		t.Fatalf("provider_auth phase with IncludeRecoveredUpstream must expose recovered rows\nfull: %s", whereProviderAuth)
	}
	if !strings.Contains(whereProviderAuth, "e.error_phase = $") {
		t.Fatalf("provider_auth recovered filter must retain its explicit phase\nfull: %s", whereProviderAuth)
	}

	whereProviderHealth, _ := buildOpsErrorLogsWhere(&ops.OpsErrorLogFilter{
		ErrorPhasesAny:           []string{"upstream", "provider_auth"},
		IncludeRecoveredUpstream: true,
	})
	if strings.Contains(whereProviderHealth, "COALESCE(e.status_code, 0) >= 400") {
		t.Fatalf("provider-health ANY filter must expose recovered inference and credential rows\nfull: %s", whereProviderHealth)
	}
	if !strings.Contains(whereProviderHealth, "e.error_phase = ANY($") {
		t.Fatalf("provider-health filter must preserve distinct phase values\nfull: %s", whereProviderHealth)
	}

	whereUserProviderAuth, _ := buildOpsErrorLogsWhere(&ops.OpsErrorLogFilter{ErrorPhasesAny: []string{"provider_auth"}})
	if !strings.Contains(whereUserProviderAuth, "COALESCE(e.status_code, 0) >= 400") {
		t.Fatalf("request-error provider_auth filters must exclude recovered successes\nfull: %s", whereUserProviderAuth)
	}

	whereMixed, _ := buildOpsErrorLogsWhere(&ops.OpsErrorLogFilter{
		ErrorPhasesAny:           []string{"provider_auth", "request"},
		IncludeRecoveredUpstream: true,
	})
	if !strings.Contains(whereMixed, "COALESCE(e.status_code, 0) >= 400") {
		t.Fatalf("recovered opt-in must not bypass the guard for non-provider phases\nfull: %s", whereMixed)
	}
}

func TestBuildOpsErrorLogsWhere_UserOwnershipIsDirectOnly(t *testing.T) {
	uid := int64(42)
	filter := &ops.OpsErrorLogFilter{UserID: &uid}
	where, args := buildOpsErrorLogsWhere(filter)
	if !strings.Contains(where, "e.user_id = $1") {
		t.Fatalf("user scope should match user_id exactly, got: %s", where)
	}
	if len(args) != 1 || args[0] != uid {
		t.Fatalf("expected user id arg %d, got %v", uid, args)
	}
	if strings.Contains(where, "deleted_key_owner_user_id") {
		t.Fatalf("user ownership must not depend on deleted-key attribution: %s", where)
	}
}

// 旧日志阶段保留在数据库中，新筛选须同时覆盖旧值。
func TestProviderAuthFilterIncludesLegacyRows(t *testing.T) {
	where, args := buildOpsErrorLogsWhere(&ops.OpsErrorLogFilter{Phase: "provider_auth"})
	if !strings.Contains(where, " OR e.error_phase = $") {
		t.Fatalf("missing legacy phase condition: %s", where)
	}
	found := false
	for _, arg := range args {
		if value, ok := arg.(string); ok && value == "account_auth" {
			found = true
		}
	}
	if !found {
		t.Fatal("legacy phase missing from query arguments")
	}
}
