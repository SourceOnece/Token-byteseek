package app

import (
	"context"
	"log/slog"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	"github.com/TokenFlux/TokenRouter/internal/routing"
)

// catalogueReader 只保留原查询范围，平台模型判断由网关适配器提供。
func catalogueReader(store *providerpostgres.ProviderStore) func(context.Context, *int64) ([]routing.CatalogueProvider, error) {
	return func(ctx context.Context, id *int64) ([]routing.CatalogueProvider, error) {
		var values []provider.Record
		var err error
		if id != nil {
			values, err = store.ListSchedulableByGroupID(ctx, *id)
		} else {
			values, err = store.ListSchedulable(ctx)
		}
		if err != nil {
			return nil, err
		}
		return gatewayprovider.CatalogueProviders(values), nil
	}
}

// provideRequestableCatalogue 与市场、模型列表共享原缓存和同一提供商存储，不经过旧网关查询。
func provideRequestableCatalogue(models *routing.ModelList, store *providerpostgres.ProviderStore, modelConfigs *routing.PricingConfigService) *routing.RequestableCatalogue {
	return &routing.RequestableCatalogue{Models: models, Read: catalogueReader(store), Resolver: routing.RequestableResolver{GroupPolicies: modelConfigs, Defaults: gatewayprovider.CatalogueDefaults(), Warn: slog.Warn}, Warn: slog.Warn}
}
