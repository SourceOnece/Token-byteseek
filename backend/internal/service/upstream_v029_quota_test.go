package service

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// 额度快照已旧但明确重置时间仍在未来时，维持模型资格暂停。
func TestOpenAIThresholdCandidate_StaleSnapshotFutureReset(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name  string
		reset map[string]any
		want  bool
	}{
		{"absolute future", map[string]any{"codex_5h_reset_at": now.Add(time.Hour).Format(time.RFC3339)}, true},
		{"relative future", map[string]any{"codex_5h_reset_after_seconds": 4 * 3600}, true},
		{"missing", nil, false},
		{"invalid", map[string]any{"codex_5h_reset_at": "invalid"}, false},
		{"past", map[string]any{"codex_5h_reset_at": now.Add(-time.Minute).Format(time.RFC3339)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extra := map[string]any{"codex_usage_updated_at": now.Add(-3 * time.Hour).Format(time.RFC3339), "codex_5h_used_percent": 99.0}
			for key, value := range tc.reset {
				extra[key] = value
			}
			candidate := openAIThresholdCandidate(extra, "5h", now)
			require.Equal(t, tc.want, candidate != nil)
			if tc.want {
				decision := EvaluateAccountSchedulingThreshold(&Account{Platform: PlatformOpenAI, Extra: extra}, map[string]int{PlatformOpenAI: 95}, now)
				require.True(t, decision.ShouldPause)
				require.NotNil(t, decision.Until)
				require.True(t, now.Before(*decision.Until))
			}
		})
	}
}

func TestResolveOpenAIQuotaUtilization_StaleSnapshotFutureReset(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name  string
		reset map[string]any
		want  bool
	}{
		{"absolute future", map[string]any{"codex_5h_reset_at": now.Add(time.Hour).Format(time.RFC3339)}, true},
		{"relative future", map[string]any{"codex_5h_reset_after_seconds": 4 * 3600}, true},
		{"missing", nil, false},
		{"invalid", map[string]any{"codex_5h_reset_at": "invalid"}, false},
		{"past", map[string]any{"codex_5h_reset_at": now.Add(-time.Minute).Format(time.RFC3339)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extra := map[string]any{"codex_usage_updated_at": now.Add(-3 * time.Hour).Format(time.RFC3339), "codex_5h_used_percent": 99.0}
			for key, value := range tc.reset {
				extra[key] = value
			}
			utilization, ok := resolveOpenAIQuotaUtilization(extra, "5h", now)
			require.Equal(t, tc.want, ok)
			if ok {
				require.Equal(t, 0.99, utilization)
			}
		})
	}
}
