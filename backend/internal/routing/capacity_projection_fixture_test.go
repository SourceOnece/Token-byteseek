// 容量测试只组合原生记录、只读查询与计数端口，规则由 routing/provider 持有。
package routing_test

import (
	"context"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
)

type testCapacityProviders struct {
	Repository capacityFixtureRepository
	Settings   capacitySettingsReader
}

func (r testCapacityProviders) ListSchedulableByGroupID(ctx context.Context, id int64) ([]provider.CapacitySnapshot, error) {
	values, err := r.Repository.ListSchedulableByGroupID(ctx, id)
	if err != nil {
		return nil, err
	}
	return capacitySnapshots(ctx, values, r.Settings), nil
}

type testtestCapacityProviderBatchReader interface {
	ListSchedulableCapacityByGroupIDs(context.Context, []int64) ([]provider.GroupProviderCapacityRow, error)
}
type testCapacityProviderBatch struct {
	testCapacityProviders
	Reader testtestCapacityProviderBatchReader
}

func (r testCapacityProviderBatch) ListSchedulableCapacityByGroupIDs(ctx context.Context, ids []int64) ([]routing.CapacityProviderRow, error) {
	values, err := r.Reader.ListSchedulableCapacityByGroupIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return capacityRows(ctx, values, r.Settings), nil
}

func newTestCapacityProviders(repo capacityFixtureRepository, settings capacitySettingsReader) routing.CapacityProviders {
	base := testCapacityProviders{Repository: repo, Settings: settings}
	if batch, ok := repo.(testtestCapacityProviderBatchReader); ok {
		return testCapacityProviderBatch{testCapacityProviders: base, Reader: batch}
	}
	return base
}

type testCapacityGroups struct{ routing.GroupRepository }

func (r testCapacityGroups) ListActiveIDs(ctx context.Context) ([]int64, error) {
	if actual, ok := r.GroupRepository.(interface {
		ListActiveIDs(context.Context) ([]int64, error)
	}); ok {
		return actual.ListActiveIDs(ctx)
	}
	values, err := r.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(values))
	for i := range values {
		ids = append(ids, values[i].ID)
	}
	return ids, nil
}

func newTestGroupCapacityService(providers capacityFixtureRepository, groups routing.GroupRepository, concurrency *scheduler.ConcurrencyService, sessions scheduler.SessionLimitCache, rpm scheduler.RPMCache, settings capacitySettingsReader) *routing.CapacityService {
	var counters routing.CapacityConcurrency
	if concurrency != nil {
		counters = concurrency
	}
	return routing.NewCapacityService(newTestCapacityProviders(providers, settings), testCapacityGroups{groups}, counters, sessions, rpm)
}

// 容量夹具只提供本组原断言实际读取的两个投影，不重建旧仓储接口。
type capacityFixtureRepository interface {
	ListSchedulableByGroupID(context.Context, int64) ([]provider.Record, error)
}
type capacitySettingsReader interface {
	GetOpenAIQuotaAutoPauseSettings(context.Context) provider.QuotaAutoPauseSettings
}

func capacitySettings(ctx context.Context, reader capacitySettingsReader) provider.QuotaAutoPauseSettings {
	if reader != nil {
		return reader.GetOpenAIQuotaAutoPauseSettings(ctx)
	}
	return provider.QuotaAutoPauseSettings{}
}

func capacitySnapshots(ctx context.Context, values []provider.Record, settings capacitySettingsReader) []provider.CapacitySnapshot {
	if len(values) == 0 {
		return nil
	}
	snapshot := capacitySettings(ctx, settings)
	out := make([]provider.CapacitySnapshot, len(values))
	for i, a := range values {
		out[i] = provider.ProjectObservedCapacity(provider.GroupProviderCapacityRow{ProviderID: a.ID, Platform: a.Platform, Concurrency: a.Concurrency, Extra: a.Extra, SessionWindowStart: a.SessionWindowStart, SessionWindowEnd: a.SessionWindowEnd}, snapshot, time.Now())
	}
	return out
}

func capacityRows(ctx context.Context, rows []provider.GroupProviderCapacityRow, settings capacitySettingsReader) []routing.CapacityProviderRow {
	if len(rows) == 0 {
		return nil
	}
	snapshot := capacitySettings(ctx, settings)
	out := make([]routing.CapacityProviderRow, len(rows))
	for i, row := range rows {
		out[i] = routing.CapacityProviderRow{GroupID: row.GroupID, Provider: provider.ProjectObservedCapacity(row, snapshot, time.Now())}
	}
	return out
}
