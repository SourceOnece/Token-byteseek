//go:build unit

package service

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQualityScheduleValidation(t *testing.T) {
	p := CodexQualitySchedule{Name: "Plan", IntervalMinutes: 60, Config: validQualityRequest()}
	require.NoError(t, p.Validate())
	require.Equal(t, 30, p.KeepRuns)
	require.Equal(t, 120, p.Config.TimeoutSeconds)
	p.IntervalMinutes = 0
	require.Error(t, p.Validate())
	p.IntervalMinutes = 60
	p.Config.ConfirmScheduling = false
	require.Error(t, p.Validate())
}

type qualityScheduleStub struct {
	qualityRepoStub
	CodexQualityScheduleRepository
	run          *CodexQualityRun
	mu           sync.Mutex
	results      []*CodexQualityResult
	status       string
	invalidLease bool
}

func (r *qualityScheduleStub) ClaimQualitySchedule(context.Context) (*CodexQualityRun, error) {
	return r.run, nil
}
func (r *qualityScheduleStub) RenewQualitySchedule(context.Context, *CodexQualityRun) (bool, error) {
	return !r.invalidLease, nil
}
func (r *qualityScheduleStub) SaveQualityRunResult(_ context.Context, id int64, result *CodexQualityResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results = append(r.results, result)
	return nil
}
func (r *qualityScheduleStub) FinishQualityRun(_ context.Context, _ *CodexQualityRun, status string) error {
	r.status = status
	return nil
}

func TestQualityScheduleRunnerRecordsAllAndReleasesBatch(t *testing.T) {
	config := validQualityRequest()
	config.AccountIDs = []int64{1, 2, 3}
	config.Concurrency = 2
	r := &qualityScheduleStub{qualityRepoStub: qualityRepoStub{account: &Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth}}, run: &CodexQualityRun{ID: 11, Config: config}}
	s := &AccountTestService{accountRepo: r}
	s.RunDueQualitySchedule(context.Background())
	require.Len(t, r.results, 3)
	for _, result := range r.results {
		require.Equal(t, "skipped", result.Status)
	}
	require.Equal(t, "completed", r.status)
	require.True(t, s.BeginCodexQualityBatch())
	s.EndCodexQualityBatch()
}

func TestQualityScheduleCorruptConfigDoesNotRun(t *testing.T) {
	r := &qualityScheduleStub{run: &CodexQualityRun{ID: 11}}
	s := &AccountTestService{accountRepo: r}
	s.RunDueQualitySchedule(context.Background())
	require.Equal(t, "failed", r.status)
	require.Empty(t, r.results)
}

func TestQualitySchedulePausedBeforeDispatchDoesNotTestAccounts(t *testing.T) {
	config := validQualityRequest()
	config.Concurrency = 1
	r := &qualityScheduleStub{run: &CodexQualityRun{ID: 11, Config: config}, invalidLease: true}
	s := &AccountTestService{accountRepo: r}
	s.RunDueQualitySchedule(context.Background())
	require.Equal(t, "interrupted", r.status)
	require.Empty(t, r.results)
}
