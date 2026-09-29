package repository

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	"github.com/TokenFlux/TokenRouter/internal/service"
)

// 状态通知沿用原outbox去重及账号缓存，用原生仓储传来的同一事务执行，禁止另开连接。
func (r *accountRepository) providerWrites() *providerpostgres.Store {
	return providerpostgres.NewWriteStore(r.client, r.sql, providerpostgres.WriteHooks{
		Publish: func(ctx context.Context, exec providerpostgres.Executor, event string, id *int64, payload any) error {
			return enqueueSchedulerOutbox(ctx, exec, event, id, nil, payload)
		},
		SyncOne:  r.syncSchedulerAccountSnapshot,
		SyncMany: r.syncSchedulerAccountSnapshots,
		NotFound: service.ErrAccountNotFound,
	})
}

// ProviderWriteStore 暴露Provider管理需要的完整写入端口；服务层仍通过接口使用，测试替身可继续走旧适配器。
func (r *accountRepository) ProviderWriteStore() provider.WriteStore { return r.providerWrites() }
