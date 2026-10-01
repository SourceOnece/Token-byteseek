package pricingcontract

import (
	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/modelidentity"
	"github.com/TokenFlux/TokenRouter/internal/modelcatalog/provider"
)

// catalogFixture 显式构造尚未启动的目录输入，不复制任何生产算法或运行状态。
type catalogFixture struct {
	pricingData map[string]*pricing.CatalogModelPricing
}

func newCatalogFixture(fixture catalogFixture) *provider.Service {
	return provider.NewServiceFromSnapshot(provider.Options{
		ModelLookupCandidates: modelidentity.CandidatesFactory,
	}, nil, provider.Snapshot{Data: fixture.pricingData})
}
