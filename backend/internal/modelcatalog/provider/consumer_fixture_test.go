package provider

import (
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	billingprovider "github.com/TokenFlux/TokenRouter/internal/billing/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/modelidentity"
	"github.com/stretchr/testify/require"
)

// 测试准备发生在启动前，直接使用所属模块状态，不复制旧服务或增加生产接口。
type modelCatalogFixture struct {
	options     Options
	pricingData map[string]*pricing.CatalogModelPricing
}

func newModelCatalogFixture(fixture modelCatalogFixture) *Service {
	options := fixture.options
	options.ModelLookupCandidates = modelidentity.CandidatesFactory
	return NewServiceFromSnapshot(options, nil, Snapshot{Data: fixture.pricingData})
}

func setPricingFixtureData(service *Service, data map[string]*pricing.CatalogModelPricing) {
	service.pricingData = data
}

func mutatePricingFixture(service *Service, change func(map[string]*pricing.CatalogModelPricing)) {
	data := service.Snapshot().Data
	change(data)
	setPricingFixtureData(service, data)
}

func setPricingFixtureRemote(service *Service, remote RemoteClient) {
	service.remoteClient = remote
}

func newStubCatalogFromJSON(t *testing.T, body string) *Service {
	t.Helper()
	service := newModelCatalogFixture(modelCatalogFixture{})
	data, err := parsePricingFixture([]byte(body))
	require.NoError(t, err)
	setPricingFixtureData(service, data)
	return service
}

// newBillingFixture 目录和计费测试组合生产计算器，并注入缺省倍率和时钟。
func newBillingFixture(catalog *Service) *billing.Calculator {
	var source billing.PriceCatalog
	if catalog != nil {
		source = catalog
	}
	warnings := &billingprovider.PricingWarnings{}
	return billing.NewCalculator(source, billing.CalculatorOptions{
		ModelPolicy:     modelidentity.PricingPolicy,
		Now:             time.Now,
		LoadLocation:    billingprovider.LoadPricingLocation,
		FallbackWarning: warnings.Fallback,
	})
}

// parsePricingFixture 纯价格测试使用生产解析器，不启动目录服务。
func parsePricingFixture(body []byte) (map[string]*pricing.CatalogModelPricing, error) {
	raw, err := pricing.DecodeCatalogEntries(body)
	if err != nil {
		return nil, err
	}
	values, diagnostics, err := pricing.ParsePricingEntries(raw)
	if validationErr := diagnostics.ValidationError(); validationErr != nil {
		return nil, validationErr
	}
	return values, err
}

// newOfflinePricingFixture 使用真实内嵌目录和发布补充文件验证默认价格。
func newOfflinePricingFixture(t *testing.T) *Service {
	t.Helper()
	service := newModelCatalogFixture(modelCatalogFixture{options: Options{DataDir: t.TempDir(), FallbackFile: "../../../resources/model-pricing/model_pricing_supplements.json"}})
	require.NoError(t, service.Initialize())
	return service
}
