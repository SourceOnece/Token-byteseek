package app

import (
	"github.com/TokenFlux/TokenRouter/internal/apikey"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/modelidentity"
	"github.com/TokenFlux/TokenRouter/internal/modelcatalog/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/postgres"
)

// provideModelAttributes 仅向管理和展示消费者提供属性服务，不注入请求执行器。
func provideModelAttributes(repo *postgres.ModelAttributeStore, catalog *provider.Service, invalidator apikey.APIKeyAuthCacheInvalidator) *routing.ModelAttributeService {
	return &routing.ModelAttributeService{
		Repo:        repo,
		Invalidator: invalidator,
		Catalog: routing.ModelAttributeCatalog{
			Lookup:     catalog.ModelAttributes,
			Update:     catalog.ForceUpdate,
			Candidates: modelidentity.CandidatesFactory,
			Snapshot: func() routing.ModelAttributeSnapshot {
				snapshot := catalog.AttributesSnapshot()
				return routing.ModelAttributeSnapshot{Items: snapshot.Items, Version: snapshot.Version, LastUpdated: snapshot.LastUpdated, LastError: snapshot.LastError}
			},
		},
	}
}
