package repository

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/stretchr/testify/require"
)

func TestCodexQualityLeaseAndAtomicWrite(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := newAccountRepositoryWithSQL(nil, db, nil)
	mock.ExpectExec(`(?s)INSERT INTO codex_quality_tests.*WHERE codex_quality_tests.lease_until <= NOW`).WithArgs(int64(1), "run", 150).WillReturnResult(sqlmock.NewResult(0, 1))
	ok, err := repo.AcquireCodexQualityTest(context.Background(), 1, "run", 120)
	require.NoError(t, err)
	require.True(t, ok)
	result := &service.CodexQualityResult{AccountID: 1, Status: "full"}
	saved := *result
	saved.SchedulingApplied = true
	saved.Schedulable = true
	raw, _ := json.Marshal(saved)
	now := time.Now()
	mock.ExpectQuery(`(?s)WITH schedule_guard AS MATERIALIZED.*updated_at=\$4.*INSERT INTO scheduler_outbox.*SELECT result FROM recorded`).
		WithArgs(int64(1), "run", true, now, "full", sqlmock.AnyArg(), service.SchedulerOutboxEventAccountChanged, int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(raw))
	ok, err = repo.FinishCodexQualityTest(context.Background(), &service.Account{ID: 1, UpdatedAt: now}, "run", result)
	require.NoError(t, err)
	require.True(t, ok)
	require.True(t, result.Schedulable)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQualityScheduleDeleteTargetsOnlyPlan(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	r := newAccountRepositoryWithSQL(nil, db, nil)
	for _, count := range []int64{1, 0} {
		mock.ExpectExec(`^DELETE FROM codex_quality_schedules WHERE id=\$1$`).WithArgs(int64(7)).WillReturnResult(sqlmock.NewResult(0, count))
		deleted, err := r.DeleteQualitySchedule(context.Background(), 7)
		require.NoError(t, err)
		require.Equal(t, count > 0, deleted)
	}
	mock.ExpectExec(`^DELETE FROM codex_quality_schedules WHERE id=\$1$`).WithArgs(int64(7)).WillReturnError(errors.New("db unavailable"))
	deleted, err := r.DeleteQualitySchedule(context.Background(), 7)
	require.Error(t, err)
	require.False(t, deleted)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQualityScheduleTriggerIsBoundedToIdlePlan(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := newAccountRepositoryWithSQL(nil, db, nil)
	mock.ExpectExec(`(?s)UPDATE codex_quality_schedules.*manual_requested_at=COALESCE.*WHERE id=\$1 AND \(active_run_id IS NULL OR lease_until<NOW\(\)\)`).WithArgs(int64(3)).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.TriggerQualitySchedule(context.Background(), 3))
	mock.ExpectExec(`(?s)UPDATE codex_quality_schedules.*WHERE id=\$1 AND \(active_run_id IS NULL OR lease_until<NOW\(\)\)`).WithArgs(int64(3)).WillReturnResult(sqlmock.NewResult(0, 0))
	require.Error(t, repo.TriggerQualitySchedule(context.Background(), 3))
	require.NoError(t, mock.ExpectationsWereMet())
}
