package completion_test

import (
	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/modelidentity"
	"github.com/TokenFlux/TokenRouter/internal/modelcatalog/provider"
)

// modelCatalogFixture 显式构造尚未启动的目录输入，不复制任何生产算法或运行状态。
type modelCatalogFixture struct {
	pricingData map[string]*pricing.CatalogModelPricing
}

func newModelCatalogFixture(fixture modelCatalogFixture) *provider.Service {
	return provider.NewServiceFromSnapshot(provider.Options{
		ModelLookupCandidates: modelidentity.CandidatesFactory,
	}, nil, provider.Snapshot{Data: fixture.pricingData})
}
