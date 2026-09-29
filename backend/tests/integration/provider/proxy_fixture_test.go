package provider_test

import (
	"context"

	_ "github.com/TokenFlux/TokenRouter/ent/runtime"

	dbent "github.com/TokenFlux/TokenRouter/ent"
	egresspostgres "github.com/TokenFlux/TokenRouter/internal/egress/postgres"
	postgresinfra "github.com/TokenFlux/TokenRouter/internal/infra/postgres"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	schedulerpostgres "github.com/TokenFlux/TokenRouter/internal/scheduler/postgres"
)

// 代理身份改变仍由同连接提供商参与者清理快照，不另开提交。
func newProxyStoreContract(client *dbent.Client, exec postgresinfra.Executor) *egresspostgres.ProxyStore {
	return egresspostgres.NewProxyStore(client, exec, egresspostgres.ProxyStoreOptions{
		Providers: func(tx postgresinfra.Executor) egresspostgres.ProxyProviderParticipant {
			return providerpostgres.ProxyChangesInTx(tx)
		},
		Enqueue: func(ctx context.Context, tx postgresinfra.Executor, payload any) error {
			return schedulerpostgres.EnqueueSchedulerChange(ctx, tx, scheduler.SchedulerOutboxEventProviderBulkChanged, nil, nil, payload)
		},
	})
}
