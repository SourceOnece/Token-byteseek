package billing_test

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/modelidentity"
	"github.com/TokenFlux/TokenRouter/internal/modelcatalog/provider"
	"github.com/stretchr/testify/require"
)

// newByteSeekOfflineCatalog 在登记的适配夹具中读取实际嵌入目录，不增加核心 I/O 依赖。
func newByteSeekOfflineCatalog(t *testing.T) *provider.Service {
	t.Helper()
	catalog := provider.NewService(provider.Options{DataDir: t.TempDir()}, nil)
	require.NoError(t, catalog.Initialize())
	return catalog
}

// catalogFixture 显式构造尚未启动的目录输入，不复制任何生产算法或运行状态。
type catalogFixture struct {
	pricingData map[string]*pricing.CatalogModelPricing
}

func newCatalogFixture(fixture catalogFixture) *provider.Service {
	return provider.NewServiceFromSnapshot(provider.Options{
		ModelLookupCandidates: modelidentity.CandidatesFactory,
	}, nil, provider.Snapshot{Data: fixture.pricingData})
}
