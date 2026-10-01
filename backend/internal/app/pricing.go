package app

import (
	"context"
	"log/slog"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	billingadapter "github.com/TokenFlux/TokenRouter/internal/billing/provider"
	"github.com/TokenFlux/TokenRouter/internal/config"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/modelidentity"
	catalogprovider "github.com/TokenFlux/TokenRouter/internal/modelcatalog/provider"
	"github.com/TokenFlux/TokenRouter/internal/pkg/timezone"
	"github.com/TokenFlux/TokenRouter/internal/routing"
)

// provideBillingCalculator 用显式配置投影构造唯一计费实例。
func provideBillingCalculator(cfg *config.Config, catalog *catalogprovider.Service, calendar timezone.Calendar) *billing.Calculator {
	warnings := &billingadapter.PricingWarnings{}
	return billing.NewCalculator(catalog, billing.CalculatorOptions{DefaultRateMultiplier: cfg.Default.RateMultiplier, ModelPolicy: modelidentity.PricingPolicy, Now: calendar.Now, LoadLocation: billingadapter.LoadPricingLocation, FallbackWarning: warnings.Fallback})
}

func provideBillingPriceResolver(modelConfigs *routing.PricingConfigService, calculator *billing.Calculator) *billing.PriceResolver {
	return billing.NewPriceResolver(modelConfigs, calculator, modelidentity.Identity, func(model string, err error) {
		slog.DebugContext(context.Background(), "failed to get model pricing from model catalog, using fallback", "model", model, "error", err)
	}, gatewayprovider.ProviderStatsSource{Service: modelConfigs})
}
