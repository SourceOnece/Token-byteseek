package app

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/TokenFlux/TokenRouter/internal/gateway/provider"

	routingprovider "github.com/TokenFlux/TokenRouter/internal/routing/provider"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/TokenFlux/TokenRouter/internal/scheduler/policy"
	settingscore "github.com/TokenFlux/TokenRouter/internal/settings"

	"github.com/TokenFlux/TokenRouter/internal/apikey"

	dbent "github.com/TokenFlux/TokenRouter/ent"

	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"

	apikeypostgres "github.com/TokenFlux/TokenRouter/internal/apikey/postgres"

	identitypostgres "github.com/TokenFlux/TokenRouter/internal/identity/postgres"

	postgresinfra "github.com/TokenFlux/TokenRouter/internal/infra/postgres"

	"github.com/TokenFlux/TokenRouter/internal/routing"

	routingpostgres "github.com/TokenFlux/TokenRouter/internal/routing/postgres"

	schedulerpostgres "github.com/TokenFlux/TokenRouter/internal/scheduler/postgres"
)

// provideRoutingGroupStore 为新旧读取入口持有唯一存储及同连接参与工厂。
func provideRoutingGroupStore(client *dbent.Client, db *sql.DB) *routingpostgres.GroupStore {
	return routingpostgres.NewGroupStore(client, db, routingpostgres.GroupStoreOptions{
		Providers: func(exec postgresinfra.Executor) routingpostgres.GroupLinkParticipant {
			return providerpostgres.GroupLinksInTx(exec)
		},
		Users: func(exec postgresinfra.Executor) routingpostgres.GroupAccessParticipant {
			return identitypostgres.GroupAccessDeletionInTx(exec)
		},
		Enqueue: func(ctx context.Context, exec postgresinfra.Executor, id *int64) error {
			return schedulerpostgres.EnqueueSchedulerChange(ctx, exec, scheduler.SchedulerOutboxEventGroupChanged, nil, id, nil)
		},
	})
}

func provideGroupReader(store *routingpostgres.GroupStore) routing.GroupRepository {
	return store
}

func provideRoutingGroupAdmin(store *routingpostgres.GroupStore, providers *providerpostgres.ProviderStore, keys *apikeypostgres.KeyStore, invalidator apikey.APIKeyAuthCacheInvalidator, modelConfigs *routing.PricingConfigService, settings *settingscore.Store, defaults *scheduler.AdminDefaults) *routing.GroupAdmin {
	return routing.NewGroupAdmin(store, store, store, routingGroupProviders{Store: providers}, keys, invalidator, modelConfigs, routing.GroupAdminOptions{
		DefaultModels: routingprovider.DefaultGroupModelCandidates,
		ModelResolver: routing.RequestableResolver{
			GroupPolicies: modelConfigs,
			Defaults:      provider.CatalogueDefaults(),
			Warn:          slog.Warn,
		},
		GlobalWeights: func(ctx context.Context) (policy.ScoreWeights, error) {
			return scheduler.LoadValidationWeights(ctx, settings, *defaults)
		}, Mutate: store.Mutate,
	})
}
