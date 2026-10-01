package billing_test

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/TokenFlux/TokenRouter/internal/billing/testkit"
	"github.com/stretchr/testify/require"
)

// TestEmptyCatalogNeverRecreatesHistoricalPrices 覆盖原静态表全部型号，防止在其他入口恢复回退。
func TestEmptyCatalogNeverRecreatesHistoricalPrices(t *testing.T) {
	calculator := billing.NewCalculator(nil, billing.CalculatorOptions{})
	for model := range testkit.HistoricalPrices() {
		price, err := calculator.GetModelPricing(model)
		require.ErrorIs(t, err, pricing.ErrModelPricingUnavailable, model)
		require.Nil(t, price, model)
		require.Equal(t, "unpriced", calculator.DefaultModelPrice(model, "", "token").PriceStatus, model)
	}
}
