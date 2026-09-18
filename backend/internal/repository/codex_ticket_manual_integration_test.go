//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/TokenFlux/TokenRouter/migrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// 真实数据库验证历史、分页、取消、失联读取；不运行真实上游采集。
func TestCodexTicketManualHistoryRealDatabase(t *testing.T) {
	ctx := context.Background()
	r := &accountRepository{sql: integrationDB}
	run := &service.CodexTicketManualRun{ID: uuid.NewString(), Config: service.CodexTicketSettings{TargetLength: 332, MaxAttempts: 200}, Total: 2, StartedAt: time.Now().UTC(), Counts: map[string]int{}}
	require.NoError(t, r.CreateTicketRun(ctx, run))
	// 幂等重放本批新增迁移必须保留已有历史，不修改旧 schema。
	raw, err := migrations.FS.ReadFile("275_codex_ticket_manual_runs.sql")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, string(raw))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM codex_ticket_manual_runs WHERE id=$1`, run.ID)
	})
	for i := 0; i < 52; i++ {
		e := &service.CodexTicketAttempt{Kind: "attempt", AccountID: 42, Model: "gpt-6-astra", Status: "missing", Attempt: i + 1, TargetLength: 332}
		require.NoError(t, r.AppendTicketEvent(ctx, run.ID, e))
		require.Positive(t, e.ID)
	}
	result := &service.CodexTicketAttempt{Kind: "result", AccountID: 42, Model: "gpt-6-astra", Status: "ready", Attempt: 52, TargetLength: 332}
	require.NoError(t, r.AppendTicketEvent(ctx, run.ID, result))
	items, total, err := r.ListTicketEvents(ctx, run.ID, "attempt", "", 42, "gpt-6-astra", 1)
	require.NoError(t, err)
	require.Len(t, items, 50)
	require.Equal(t, 52, total)
	items, total, err = r.ListTicketEvents(ctx, run.ID, "attempt", "", 42, "gpt-6-astra", 2)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, 52, total)
	items, total, err = r.ListTicketEvents(ctx, run.ID, "result", "ready", 0, "", 1)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, 1, total)
	loaded, err := r.GetTicketRun(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, 1, loaded.Counts["ready"])
	require.Equal(t, 332, loaded.Config.TargetLength)
	_, err = integrationDB.ExecContext(ctx, `UPDATE codex_ticket_manual_runs SET heartbeat_at=NOW()-INTERVAL '2 minutes' WHERE id=$1`, run.ID)
	require.NoError(t, err)
	loaded, err = r.GetTicketRun(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, "interrupted", loaded.Status)
	require.NoError(t, r.HeartbeatTicketRun(ctx, run.ID))
	loaded, err = r.GetTicketRun(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, "running", loaded.Status)
	require.NoError(t, r.FinishTicketRun(ctx, run.ID, "cancelled", map[string]int{"ready": 1}))
	loaded, err = r.GetTicketRun(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, "cancelled", loaded.Status)
	require.NotNil(t, loaded.FinishedAt)
	require.Error(t, r.AppendTicketEvent(ctx, run.ID, result))
	runs, err := r.ListTicketRuns(ctx, 1)
	require.NoError(t, err)
	require.NotEmpty(t, runs)
}
