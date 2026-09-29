package postgres

import (
	"context"
	"database/sql"
	"errors"

	dbent "github.com/TokenFlux/TokenRouter/ent"
)

// Executor 使原生仓储在生产与SQL测试中共用执行边界；回调必须使用传入的当前事务。
type Executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// WriteHooks 暂由旧组合层绑定调度发布和缓存，避免迁移时创建第二个outbox或缓存。
// Publish必须在传入事务写入，不能转回连接池或异步提交。
type WriteHooks struct {
	Publish  func(context.Context, Executor, string, *int64, any) error
	SyncOne  func(context.Context, int64)
	SyncMany func(context.Context, []int64)
	NotFound error
}

func NewWriteStore(client *dbent.Client, executor Executor, hooks WriteHooks) *Store {
	return &Store{client: client, sql: executor, hooks: hooks}
}

func (s *Store) publish(ctx context.Context, executor Executor, event string, id *int64, payload any) error {
	if s.hooks.Publish == nil {
		return errors.New("provider publication hook is not configured")
	}
	return s.hooks.Publish(ctx, executor, event, id, payload)
}

func (s *Store) syncOne(ctx context.Context, id int64) {
	if s.hooks.SyncOne != nil {
		s.hooks.SyncOne(ctx, id)
	}
}
func (s *Store) syncMany(ctx context.Context, ids []int64) {
	if s.hooks.SyncMany != nil {
		s.hooks.SyncMany(ctx, ids)
	}
}
func (s *Store) notFound() error {
	if s.hooks.NotFound != nil {
		return s.hooks.NotFound
	}
	return ErrNotFound
}
func (s *Store) translateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) || errors.Is(err, sql.ErrNoRows) || dbent.IsNotFound(err) {
		return s.notFound()
	}
	return err
}

func clientFromContext(ctx context.Context, client *dbent.Client) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return client
}

// 原事件枚举在过渡期保持不变，切换表名时另行迁移，不在内部改名时改变缓存消费者契约。
const (
	eventChanged     = "account_changed"
	eventBulkChanged = "account_bulk_changed"
)
