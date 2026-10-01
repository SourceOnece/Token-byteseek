package routing_test

import (
	"context"
	"log/slog"

	routingtestkit "github.com/TokenFlux/TokenRouter/internal/routing/testkit"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/modelidentity"
	"github.com/TokenFlux/TokenRouter/internal/routing"
)

type marketplaceQuoteFixture struct {
	calculator *billing.Calculator
	resolver   *billing.PriceResolver
}

func (p marketplaceQuoteFixture) Quote(ctx context.Context, req routing.MarketplaceQuoteRequest) pricing.ModelDisplayPricing {
	resolver := p.resolver
	if resolver == nil {
		resolver = billing.NewPriceResolver(nil, p.calculator, modelidentity.Identity, func(model string, err error) {
			slog.Debug("failed to get model pricing from model catalog, using fallback", "model", model, "error", err)
		})
	}
	return resolver.PublicQuote(ctx, billing.PublicQuoteInput{PricingInput: billing.PricingInput{Model: req.Model, GroupID: &req.GroupID}, RateMultiplier: req.RateMultiplier, FreeFastApplicable: req.FreeFastApplicable})
}

func (p marketplaceQuoteFixture) GetModelModalities(model string) ([]string, []string) {
	return p.calculator.GetModelModalities(model)
}

// marketplaceWithConfig 将公开报价合同绑定到真实共享配置解析。
func marketplaceWithConfig(calculator *billing.Calculator, groupID int64, settings pricing.BillingSettings, cards []routing.ModelPricingEntry) *routing.Marketplace {
	configs := routingtestkit.ModelConfigFromData(routingtestkit.ModelConfigDataFromRows([]routingtestkit.Configuration{{ID: 1, Status: routing.StatusActive, GroupIDs: []int64{groupID}, BillingSettings: &settings, ModelPricing: cards}}, nil))
	return newMarketplaceFixture(nil, nil, calculator, NewModelPricingResolver(configs, calculator))
}
