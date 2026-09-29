package usage

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type blockingAggregation struct {
	DashboardAggregationRepository
	entered  chan struct{}
	release  chan struct{}
	canceled atomic.Bool
}

func (r *blockingAggregation) RecomputeRange(ctx context.Context, a, b time.Time) error {
	close(r.entered)
	select {
	case <-ctx.Done():
		r.canceled.Store(true)
		return ctx.Err()
	case <-r.release:
		return nil
	}
}

func TestRegressionAggregationStopCancelsWork(t *testing.T) {
	r := &blockingAggregation{entered: make(chan struct{}), release: make(chan struct{})}
	s := NewDashboardAggregationService(r, nil, nil)
	s.runtimeStarted = true
	if e := s.TriggerRecomputeRange(time.Now().Add(-time.Hour), time.Now()); e != nil {
		t.Fatal(e)
	}
	<-r.entered
	done := make(chan struct{})
	go func() { s.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Error("Stop did not cancel context-aware in-flight aggregation")
	}
	close(r.release)
	<-done
	if !r.canceled.Load() {
		t.Error("aggregation context was never canceled")
	}
}

type partialCleanup struct {
	UsageCleanupRepository
	checks  int
	deleted int64
}

func (r *partialCleanup) GetTaskStatus(context.Context, int64) (string, error) {
	r.checks++
	if r.checks > 1 {
		return UsageCleanupStatusCanceled, nil
	}
	return UsageCleanupStatusRunning, nil
}

func (r *partialCleanup) DeleteUsageLogsBatch(context.Context, UsageCleanupFilters, int) (int64, error) {
	r.deleted = 5000
	return 5000, nil
}
func (r *partialCleanup) UpdateTaskProgress(context.Context, int64, int64) error { return nil }

type recomputeRecorder struct {
	DashboardAggregationRepository
	calls atomic.Int64
}

func (r *recomputeRecorder) RecomputeRange(context.Context, time.Time, time.Time) error {
	r.calls.Add(1)
	return nil
}

func TestRegressionCanceledCleanupRepairsAggregates(t *testing.T) {
	r := &partialCleanup{}
	ar := &recomputeRecorder{}
	agg := NewDashboardAggregationService(ar, nil, nil)
	agg.runtimeStarted = true
	s := NewUsageCleanupService(r, nil, agg, nil)
	defer s.Stop()
	s.executeTask(context.Background(), &UsageCleanupTask{ID: 1, Filters: UsageCleanupFilters{StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now().Add(-time.Hour)}})
	agg.runtimeWG.Wait()
	agg.Stop()
	if r.deleted != 5000 {
		t.Fatalf("fixture failed to delete batch")
	}
	if ar.calls.Load() == 0 {
		t.Error("5000 rows were deleted before cancellation, but neither legacy nor analytics recompute was requested")
	}
}

// 在真实运行标记之外，手动回填的全量状态写入可以覆盖并发实时进度。
type stateRepo struct {
	UsageAnalyticsAggregationRepository
	mu      sync.Mutex
	state   UsageAnalyticsAggregationState
	reads   atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func (r *stateRepo) GetUsageAnalyticsAggregationState(context.Context) (*UsageAnalyticsAggregationState, error) {
	r.mu.Lock()
	v := r.state
	r.mu.Unlock()
	if r.reads.Add(1) == 1 {
		close(r.entered)
		<-r.release
	}
	return &v, nil
}

func (r *stateRepo) SaveUsageAnalyticsAggregationState(_ context.Context, v *UsageAnalyticsAggregationState) error {
	r.mu.Lock()
	r.state = *v
	r.mu.Unlock()
	return nil
}

func TestRegressionManualBackfillPreservesLiveProgress(t *testing.T) {
	old := time.Date(2026, 9, 1, 1, 0, 0, 0, time.UTC)
	next := old.Add(time.Hour)
	r := &stateRepo{state: UsageAnalyticsAggregationState{LiveWatermark: old}, entered: make(chan struct{}), release: make(chan struct{})}
	s := NewDashboardAggregationService(&dashboardAggregationRepoTestStub{}, nil, &Options{DashboardAgg: DashboardAggregationConfig{BackfillEnabled: true}})
	s.analyticsRepo = r
	done := make(chan error, 1)
	go func() { done <- s.TriggerBackfill(old.Add(-time.Hour), old) }()
	<-r.entered
	s.markAnalyticsLiveSuccess(context.Background(), next, old, time.Now())
	close(r.release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	r.mu.Lock()
	got := r.state.LiveWatermark
	r.mu.Unlock()
	if !got.Equal(next) {
		t.Errorf("live watermark regressed: want %v got %v", next, got)
	}
}

func (r *stateRepo) ApplyUsageAnalyticsState(_ context.Context, c AnalyticsStateChange) (*UsageAnalyticsAggregationState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, e := ApplyAnalyticsStateChange(r.state, c)
	if e != nil {
		return nil, e
	}
	r.state = v
	return &v, nil
}
