//go:build integration

package provider_test

import (
	"context"

	billingpostgres "github.com/TokenFlux/TokenRouter/internal/billing/postgres"
	postgresinfra "github.com/TokenFlux/TokenRouter/internal/infra/postgres"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
)

// 资金参与夹具仅供真实存储测试使用，与消费者保持相同构建条件。
func newProviderUsageContract(exec postgresinfra.Executor, store *providerpostgres.ProviderStore, cache scheduler.SnapshotPublicationCache) *billingpostgres.ProviderUsageStore {
	events := providerPublicationEvents(store, cache)
	return billingpostgres.NewProviderUsageStore(exec, billingpostgres.ProviderUsageOptions{
		Changed: func(ctx context.Context, id int64) error {
			return events.Write(ctx, exec, providerpostgres.ProviderChanged, &id, nil, nil)
		},
		Sync: events.SyncOne,
	})
}
