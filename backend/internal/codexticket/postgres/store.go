package postgres

import (
	"context"
	"database/sql"
	"time"

	dbent "github.com/TokenFlux/TokenRouter/ent"
	"github.com/TokenFlux/TokenRouter/ent/setting"
	"github.com/TokenFlux/TokenRouter/internal/codexticket"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/TokenFlux/TokenRouter/internal/settings"
)

// 票据扩展共用唯一 Provider 仓储及连接，配置 CAS 和专用历史仍由本模块拥有。
type Store struct {
	*providerpostgres.ProviderStore
	settings.Repository
	client *dbent.Client
	sql    *sql.DB
	events providerpostgres.ProviderEvents
}

func New(client *dbent.Client, db *sql.DB, providers *providerpostgres.ProviderStore, settings settings.Repository, events providerpostgres.ProviderEvents) *Store {
	return &Store{ProviderStore: providers, Repository: settings, client: client, sql: db, events: events}
}

func (r *Store) syncSchedulerAccountSnapshot(ctx context.Context, id int64) {
	r.events.SyncOne(ctx, id)
}

func (r *Store) CompareAndSwapTicketSettings(ctx context.Context, expected *string, value string) (bool, error) {
	const key = "codex_ticket_runtime"
	if expected == nil {
		_, err := r.client.Setting.Create().SetKey(key).SetValue(value).SetUpdatedAt(time.Now()).Save(ctx)
		if dbent.IsConstraintError(err) {
			return false, nil
		}
		return err == nil, err
	}
	n, err := r.client.Setting.Update().Where(setting.KeyEQ(key), setting.ValueEQ(*expected)).SetValue(value).SetUpdatedAt(time.Now()).Save(ctx)
	return n == 1, err
}

const SchedulerOutboxEventProviderChanged = scheduler.SchedulerOutboxEventProviderChanged

// 明确选择设置删除端口，避免嵌入 Provider 的同名方法造成接口歧义。
func (r *Store) Delete(ctx context.Context, key string) error { return r.Repository.Delete(ctx, key) }

var _ codexticket.CodexTicketAccountStore = (*Store)(nil)
var _ codexticket.CodexTicketSettingsStore = (*Store)(nil)
