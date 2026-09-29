package app

import (
	"context"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/usage"
)

type providerLocalStats struct{ source usage.UsageLogRepository }

func localWindowStats(v *usage.ProviderStats) *provider.WindowStats {
	if v == nil {
		return nil
	}
	return &provider.WindowStats{Requests: v.Requests, Tokens: v.Tokens, Cost: v.Cost, StandardCost: v.StandardCost, UserCost: v.UserCost}
}

func (r providerLocalStats) GetProviderWindowStats(ctx context.Context, id int64, start time.Time) (*provider.WindowStats, error) {
	v, err := r.source.GetProviderWindowStats(ctx, id, start)
	return localWindowStats(v), err
}

func (r providerLocalStats) GetProviderTodayStats(ctx context.Context, id int64) (*provider.WindowStats, error) {
	v, err := r.source.GetProviderTodayStats(ctx, id)
	return localWindowStats(v), err
}

type providerLocalStatsBatchSource interface {
	GetProviderWindowStatsBatch(context.Context, []int64, time.Time) (map[int64]*usage.ProviderStats, error)
}
type providerLocalStatsBatch struct {
	providerLocalStats
	batch providerLocalStatsBatchSource
}

func (r providerLocalStatsBatch) GetProviderWindowStatsBatch(ctx context.Context, ids []int64, start time.Time) (map[int64]*provider.WindowStats, error) {
	values, err := r.batch.GetProviderWindowStatsBatch(ctx, ids, start)
	if values == nil {
		return nil, err
	}
	out := make(map[int64]*provider.WindowStats, len(values))
	for id, v := range values {
		out[id] = localWindowStats(v)
	}
	return out, err
}

// newProviderLocalUsageStats 投影 usage 查询；批量失败回退由 provider 拥有。
func newProviderLocalUsageStats(source usage.UsageLogRepository) provider.LocalUsageStats {
	reader := providerLocalStats{source}
	if batch, ok := source.(providerLocalStatsBatchSource); ok {
		return providerLocalStatsBatch{reader, batch}
	}
	return reader
}
