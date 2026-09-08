package repository

import (
	"context"
	"encoding/json"
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
