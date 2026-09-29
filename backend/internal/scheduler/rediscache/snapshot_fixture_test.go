// 测试只解包受控提供商记录，Redis 行为由生产 SnapshotCache 执行。
package rediscache

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/scheduler/rediscache/codec"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/redis/go-redis/v9"
)

type schedulerCache struct{ *SnapshotCache }

func NewSchedulerCache(rdb *redis.Client) *schedulerCache {
	return &schedulerCache{NewSnapshotCache(rdb, codec.ProviderCodec{})}
}
func (c *schedulerCache) SnapshotCoreCache() scheduler.SnapshotCache { return c.SnapshotCache }
func (c *schedulerCache) GetSnapshot(ctx context.Context, bucket scheduler.SchedulerBucket) ([]*providercore.Record, bool, error) {
	values, hit, err := c.SnapshotCache.GetSnapshot(ctx, bucket)
	if err != nil {
		return nil, hit, err
	}
	if values == nil {
		return nil, hit, nil
	}
	out := make([]*providercore.Record, len(values))
	for i, v := range values {
		out[i], err = codec.RecordValue(v)
		if err != nil {
			return nil, false, err
		}
	}
	return out, hit, nil
}

func (c *schedulerCache) GetProvider(ctx context.Context, id int64) (*providercore.Record, error) {
	v, err := c.SnapshotCache.GetProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	return codec.RecordValue(v)
}

func (c *schedulerCache) SetProvider(ctx context.Context, v *providercore.Record) error {
	return c.SnapshotCache.SetProvider(ctx, codec.WrapRecord(v))
}

func (c *schedulerCache) SetSnapshot(ctx context.Context, bucket scheduler.SchedulerBucket, token scheduler.SchedulerBucketWriteToken, values []providercore.Record) error {
	return c.SnapshotCache.SetSnapshot(ctx, bucket, token, snapshotRecords(values))
}

func (c *schedulerCache) SetSnapshotAndReturnProviderIDs(ctx context.Context, bucket scheduler.SchedulerBucket, token scheduler.SchedulerBucketWriteToken, values []providercore.Record) ([]int64, error) {
	return c.SnapshotCache.SetSnapshotAndReturnProviderIDs(ctx, bucket, token, snapshotRecords(values))
}

func buildSchedulerMetadataProvider(value providercore.Record) providercore.Record {
	return (codec.ProviderCodec{}).Metadata(value)
}

func snapshotRecords(values []providercore.Record) []scheduler.SnapshotProvider {
	if values == nil {
		return nil
	}
	out := make([]scheduler.SnapshotProvider, len(values))
	for i := range values {
		out[i] = codec.WrapRecord(&values[i])
	}
	return out
}

func filterSchedulerCredentials(value map[string]any) map[string]any {
	return buildSchedulerMetadataProvider(providercore.Record{Credentials: value}).Credentials
}
