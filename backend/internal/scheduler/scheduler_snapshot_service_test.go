package scheduler

import (
	"context"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	"github.com/stretchr/testify/require"
)

type schedulerSnapshotContextCacheStub struct {
	SnapshotCache
}

func (s schedulerSnapshotContextCacheStub) GetSnapshot(ctx context.Context, bucket SchedulerBucket) ([]SnapshotProvider, bool, error) {
	return nil, false, ctx.Err()
}

type schedulerSnapshotFallbackRepoStub struct {
	SnapshotProviderSource
	calls int
}

func (r *schedulerSnapshotFallbackRepoStub) ListSchedulableByPlatform(ctx context.Context, platform string) ([]SnapshotProvider, error) {
	r.calls++
	return nil, nil
}

func (r *schedulerSnapshotFallbackRepoStub) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]SnapshotProvider, error) {
	r.calls++
	return nil, nil
}

func TestSchedulerSnapshotService_ListSchedulableProvidersStopsWhenCacheContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo := &schedulerSnapshotFallbackRepoStub{}
	svc := NewSnapshotService(schedulerSnapshotContextCacheStub{}, nil, repo, nil, nil)

	providers, useMixed, err := svc.ListSchedulableProviders(ctx, nil, capability.PlatformOpenAI, false)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, providers)
	require.False(t, useMixed)
	require.Equal(t, 0, repo.calls)
}

func TestSchedulerSnapshotPlatformsIncludesQoder(t *testing.T) {
	require.Contains(t, schedulerSnapshotPlatforms(), capability.PlatformQoder)
}

func TestSchedulerSnapshotServiceCanonicalBucketsIncludesQoder(t *testing.T) {
	buckets := schedulerCanonicalBuckets(0)

	require.Contains(t, buckets, SchedulerBucket{GroupID: 0, Platform: capability.PlatformQoder, Mode: SchedulerModeSingle})
	require.Contains(t, buckets, SchedulerBucket{GroupID: 0, Platform: capability.PlatformQoder, Mode: SchedulerModeForced})
	require.NotContains(t, buckets, SchedulerBucket{GroupID: 0, Platform: capability.PlatformQoder, Mode: SchedulerModeMixed})
}
