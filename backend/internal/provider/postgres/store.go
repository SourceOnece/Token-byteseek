// Package postgres 实现Provider记录的数据库读取，迁移阶段继续访问原accounts表。
package postgres

import (
	"context"
	"errors"
	"maps"

	dbent "github.com/TokenFlux/TokenRouter/ent"
	dbaccount "github.com/TokenFlux/TokenRouter/ent/account"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// ErrNotFound 在适配旧管理API时翻译为原ACCOUNT_NOT_FOUND，不能静默改公开错误码。
var ErrNotFound = errors.New("provider record not found")

// Store 复用传入的Ent客户端，含事务和软删除拦截器，不创建第二套连接或持有数据缓存。
type Store struct {
	client *dbent.Client
	sql    Executor
	hooks  WriteHooks
}

func NewStore(client *dbent.Client) *Store { return &Store{client: client} }

func (s *Store) GetByID(ctx context.Context, id int64) (*provider.Record, error) {
	entity, err := s.client.Account.Query().Where(dbaccount.IDEQ(id)).Only(ctx)
	if dbent.IsNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return FromAccountEntity(entity), nil
}

// GetByIDs 保留首见顺序，忽略非正数/重复/不存在ID；代理沿原批量入口预加载。
func (s *Store) GetByIDs(ctx context.Context, ids []int64) ([]*provider.Record, error) {
	unique := make([]int64, 0, len(ids))
	seen := map[int64]bool{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			unique = append(unique, id)
			seen[id] = true
		}
	}
	if len(unique) == 0 {
		return []*provider.Record{}, nil
	}
	entities, err := s.client.Account.Query().Where(dbaccount.IDIn(unique...)).WithProxy().All(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]*provider.Record, len(entities))
	for _, entity := range entities {
		byID[entity.ID] = FromAccountEntity(entity)
	}
	result := make([]*provider.Record, 0, len(entities))
	for _, id := range unique {
		if record := byID[id]; record != nil {
			result = append(result, record)
		}
	}
	return result, nil
}

// FromAccountEntity 显式映射数据库字段；不改JSON键、ID、时间精度或凭据内容。
// 与旧转换器一样复制顶层配置map，缓存私有字段由各自投影负责，不能直接Marshal Record。
func FromAccountEntity(m *dbent.Account) *provider.Record {
	if m == nil {
		return nil
	}
	rate := m.RateMultiplier
	r := &provider.Record{
		ID: m.ID, Name: m.Name, Notes: m.Notes, Platform: m.Platform, Type: m.Type,
		Credentials: maps.Clone(m.Credentials), Extra: maps.Clone(m.Extra), ProxyID: m.ProxyID,
		ProxyFallbackOriginID: m.ProxyFallbackOriginID, Concurrency: m.Concurrency, Priority: m.Priority,
		RateMultiplier: &rate, LoadFactor: m.LoadFactor, Status: m.Status,
		ErrorMessage: stringValue(m.ErrorMessage), LastUsedAt: m.LastUsedAt, ExpiresAt: m.ExpiresAt,
		AutoPauseOnExpired: m.AutoPauseOnExpired, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		Schedulable: m.Schedulable, RateLimitedAt: m.RateLimitedAt, RateLimitResetAt: m.RateLimitResetAt,
		OverloadUntil: m.OverloadUntil, TempUnschedulableUntil: m.TempUnschedulableUntil,
		TempUnschedulableReason: stringValue(m.TempUnschedulableReason), SessionWindowStart: m.SessionWindowStart,
		SessionWindowEnd: m.SessionWindowEnd, SessionWindowStatus: stringValue(m.SessionWindowStatus),
		ParentProviderID: m.ParentAccountID, QuotaDimension: string(m.QuotaDimension),
	}
	if p := m.Edges.Proxy; p != nil {
		r.Proxy = &egress.Proxy{ID: p.ID, Name: p.Name, Protocol: p.Protocol, Host: p.Host, Port: p.Port,
			Username: stringValue(p.Username), Password: stringValue(p.Password), Status: p.Status,
			CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, ExpiresAt: p.ExpiresAt, FallbackMode: p.FallbackMode,
			BackupProxyID: p.BackupProxyID, ExpiryWarnDays: p.ExpiryWarnDays}
	}
	return r
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
