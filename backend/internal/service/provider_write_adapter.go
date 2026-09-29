package service

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// providerWriteAdapter 将管理层经过校验的Provider Record交给现有事务仓储。
// 适配器不复制事务、缓存或outbox；下一阶段可替换内部实现而不改管理用例。
type providerWriteAdapter struct{ repo AccountRepository }

func (a providerWriteAdapter) Update(ctx context.Context, record *provider.Record) error {
	account := AccountFromProviderRecord(record)
	if account == nil {
		return ErrAccountNilInput
	}
	return a.repo.Update(ctx, account)
}

func (a providerWriteAdapter) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	return a.repo.UpdateExtra(ctx, id, updates)
}

func (s *adminServiceImpl) providerWriteStore() provider.WriteStore {
	return providerWriteAdapter{repo: s.accountRepo}
}
