//go:build unit

package rediscache

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/scheduler/rediscache/codec"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/redis/go-redis/v9"
)

// unit 白盒断言沿用旧提供商报文，只委托新缓存与兼容 codec。
func newSchedulerCacheWithChunkSizes(rdb *redis.Client, read, write int) *schedulerCache {
	return &schedulerCache{NewSnapshotCache(rdb, codec.ProviderCodec{}, SnapshotCacheOptions{MGetChunkSize: read, WriteChunkSize: write})}
}

func (c *schedulerCache) writeProviderIDs(ctx context.Context, values []providercore.Record) ([]int64, error) {
	return c.SnapshotCache.writeProviderIDs(ctx, snapshotRecords(values))
}

func (c *schedulerCache) writeSnapshotVersionAndReturnProviderIDs(ctx context.Context, bucket scheduler.SchedulerBucket, version string, values []providercore.Record) ([]int64, error) {
	return c.SnapshotCache.writeSnapshotVersionAndReturnProviderIDs(ctx, bucket, version, snapshotRecords(values))
}

func marshalSchedulerCacheProvider(value providercore.Record) ([]byte, []byte, error) {
	return (codec.ProviderCodec{}).Encode(codec.WrapRecord(&value))
}
