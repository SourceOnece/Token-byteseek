package rediscache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// 固定窗口不被后续创建延长，不与失败次数或其他用户共享。
func TestCreateFrequencyWindowAndIsolation(t *testing.T) {
	r := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: r.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := &ApiKeyCache{rdb: client}
	ctx := context.Background()
	count, err := cache.IncrementCreateCount(ctx, 7, time.Hour)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	r.FastForward(30 * time.Minute)
	count, err = cache.IncrementCreateCount(ctx, 7, time.Hour)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.Equal(t, 30*time.Minute, r.TTL("apikey:create_count:7"))
	count, err = cache.IncrementCreateCount(ctx, 8, time.Hour)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.NoError(t, cache.IncrementCreateAttemptCount(ctx, 7))
	r.FastForward(31 * time.Minute)
	count, err = cache.IncrementCreateCount(ctx, 7, time.Hour)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}
