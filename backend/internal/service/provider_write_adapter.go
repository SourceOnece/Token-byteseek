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
	err := a.repo.Update(ctx, account)
	record.Extra = account.Extra
	record.UpdatedAt = account.UpdatedAt
	return err
}

func (a providerWriteAdapter) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	return a.repo.UpdateExtra(ctx, id, updates)
}

func (a providerWriteAdapter) UpdateCredentials(ctx context.Context, id int64, credentials map[string]any) error {
	if updater, ok := any(a.repo).(accountCredentialsUpdater); ok {
		return updater.UpdateCredentials(ctx, id, credentials)
	}
	// 兼容旧测试替身时先读取完整记录，禁止用只含 ID/凭据的半成品覆盖
	// 名称、分组、代理、调度和票据配置。生产仓储会走专用凭据端口。
	current, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return ErrAccountNotFound
	}
	current.Credentials = credentials
	return a.repo.Update(ctx, current)
}

func (a providerWriteAdapter) BulkUpdate(ctx context.Context, ids []int64, updates provider.BulkUpdate) (int64, error) {
	return a.repo.BulkUpdate(ctx, ids, updates)
}

func (s *adminServiceImpl) providerWriteStore() provider.WriteStore {
	if factory, ok := any(s.accountRepo).(interface{ ProviderWriteStore() provider.WriteStore }); ok {
		return factory.ProviderWriteStore()
	}
	return providerWriteAdapter{repo: s.accountRepo}
}
