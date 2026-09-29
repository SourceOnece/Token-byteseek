package app

import (
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	"github.com/TokenFlux/TokenRouter/internal/routing"
)

// gatewayModelAvailability 固定三种诊断意图，共用提供商存储和分组映射读取实例，不读取执行服务。
type gatewayModelAvailability struct {
	Messages   routing.ModelAvailabilityDiagnoser
	Compatible routing.ModelAvailabilityDiagnoser
	Resolved   routing.ModelAvailabilityDiagnoser
}

func provideGatewayModelAvailability(store *postgres.ProviderStore, modelConfigs *routing.PricingConfigService) *gatewayModelAvailability {
	var source gatewayprovider.AvailabilityProviders
	if store != nil {
		source = store
	}
	general := gatewayprovider.NewModelAvailability(source, modelConfigs, false)
	compatible := gatewayprovider.NewModelAvailability(source, modelConfigs, true)
	return &gatewayModelAvailability{
		Messages:   routing.ModelAvailabilityDiagnoserFunc(general.DiagnoseGeneral),
		Compatible: routing.ModelAvailabilityDiagnoserFunc(compatible.DiagnoseCompatible),
		Resolved:   routing.ModelAvailabilityDiagnoserFunc(compatible.DiagnoseCompatibleRouting),
	}
}
