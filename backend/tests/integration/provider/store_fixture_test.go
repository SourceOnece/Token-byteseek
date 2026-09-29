package provider_test

import (
	"context"

	dbent "github.com/TokenFlux/TokenRouter/ent"
	egresspostgres "github.com/TokenFlux/TokenRouter/internal/egress/postgres"
	postgresinfra "github.com/TokenFlux/TokenRouter/internal/infra/postgres"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	"github.com/TokenFlux/TokenRouter/internal/routing/accessview"
	routingpostgres "github.com/TokenFlux/TokenRouter/internal/routing/postgres"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	schedulerpostgres "github.com/TokenFlux/TokenRouter/internal/scheduler/postgres"
	"github.com/TokenFlux/TokenRouter/internal/scheduler/rediscache/codec"
)

// 原存储契约只绑定真实 outbox writer；没有缓存时不引入额外读取。
func newProviderStoreContract(client *dbent.Client, exec postgresinfra.Executor, cache scheduler.SnapshotPublicationCache) *providerpostgres.ProviderStore {
	store := providerpostgres.NewProviderStore(client, exec, providerpostgres.ProviderStoreOptions{
		Group: func(g *dbent.Group) *accessview.GroupConfig {
			return (*accessview.GroupConfig)(routingpostgres.GroupFromEnt(g))
		},
		OllamaIdentity: provider.IsOllamaCloudUsageProvider,
		Proxy:          egresspostgres.ProxyEntity,
		Events:         providerEventsFixture{},
	})
	store.SetEvents(providerPublicationEvents(store, cache))
	return store
}

type providerEventsFixture struct{ publisher scheduler.SnapshotPublisher }

func (providerEventsFixture) Name(event providerpostgres.ProviderEvent) string {
	switch event {
	case providerpostgres.ProviderChanged:
		return scheduler.SchedulerOutboxEventProviderChanged
	case providerpostgres.ProviderGroupsChanged:
		return scheduler.SchedulerOutboxEventProviderGroupsChanged
	case providerpostgres.ProviderLastUsed:
		return scheduler.SchedulerOutboxEventProviderLastUsed
	case providerpostgres.ProviderBulkChanged:
		return scheduler.SchedulerOutboxEventProviderBulkChanged
	default:
		panic("未知提供商事件")
	}
}

func (f providerEventsFixture) Write(ctx context.Context, exec postgresinfra.Executor, event providerpostgres.ProviderEvent, id, group *int64, payload any) error {
	return schedulerpostgres.EnqueueSchedulerChange(ctx, exec, f.Name(event), id, group, payload)
}

func (providerEventsFixture) GroupPayload(ids []int64) any { return scheduler.GroupPayload(ids) }

func (f providerEventsFixture) SyncOne(ctx context.Context, id int64) { f.publisher.Publish(ctx, id) }

func (f providerEventsFixture) SyncMany(ctx context.Context, ids []int64) {
	f.publisher.PublishMany(ctx, ids)
}
func (f providerEventsFixture) Drop(ctx context.Context, id int64) { f.publisher.Drop(ctx, id) }

// 夹具只连接实际 outbox 与发布实现，不复制锁、编码或事件合并规则。
func providerPublicationEvents(store *providerpostgres.ProviderStore, cache scheduler.SnapshotPublicationCache) providerEventsFixture {
	return providerEventsFixture{publisher: scheduler.SnapshotPublisher{
		Cache: cache,
		Read: func(ctx context.Context, id int64) (scheduler.SnapshotProvider, error) {
			v, err := store.GetByID(ctx, id)
			return codec.WrapRecord(v), err
		},
		ReadMany: func(ctx context.Context, ids []int64) ([]scheduler.SnapshotProvider, error) {
			values, err := store.GetByIDs(ctx, ids)
			if values == nil {
				return nil, err
			}
			out := make([]scheduler.SnapshotProvider, len(values))
			for i, value := range values {
				out[i] = codec.WrapRecord(value)
			}
			return out, err
		},
	}}
}

func normalizeJSONMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
