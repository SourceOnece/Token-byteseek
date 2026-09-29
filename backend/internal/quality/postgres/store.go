package postgres

import (
	"context"
	"database/sql"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
)

// 质量历史独立持久化，账号状态变化沿同一个 outbox 和调度缓存发布。
type Store struct {
	*providerpostgres.ProviderStore
	sql    *sql.DB
	events providerpostgres.ProviderEvents
}

func New(data *providerpostgres.ProviderStore, db *sql.DB, events providerpostgres.ProviderEvents) *Store {
	return &Store{ProviderStore: data, sql: db, events: events}
}
func (r *Store) syncSchedulerAccountSnapshot(ctx context.Context, id int64) {
	r.events.SyncOne(ctx, id)
}

const SchedulerOutboxEventProviderChanged = scheduler.SchedulerOutboxEventProviderChanged
