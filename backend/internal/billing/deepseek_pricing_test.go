//go:build unit

package billing_test

import (
	"context"
	"testing"
	"time"

	billingtestkit "github.com/TokenFlux/TokenRouter/internal/billing/testkit"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	billingpricing "github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/routing"

	"github.com/stretchr/testify/require"
)

func TestDeepseekPeakMultiplierAt(t *testing.T) {
	weekday := func(hour, minute int) time.Time {
		return time.Date(2026, 8, 24, hour, minute, 0, 0, time.UTC)
	}
	tests := []struct {
		name string
		now  time.Time
		want float64
	}{
		{name: "weekday peak start", now: weekday(1, 0), want: 2},
		{name: "weekday peak end", now: weekday(4, 0), want: 1},
		{name: "weekday second peak", now: weekday(6, 30), want: 2},
		{name: "weekday second peak end", now: weekday(10, 0), want: 1},
		{name: "weekday off peak", now: weekday(12, 0), want: 1},
		{name: "beijing weekend", now: time.Date(2026, 8, 22, 2, 0, 0, 0, time.UTC), want: 1},
		{name: "utc weekend boundary", now: time.Date(2026, 8, 22, 16, 30, 0, 0, time.UTC), want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, billingpricing.DeepseekPeakMultiplierAt(tt.now))
		})
	}
}

func TestGetModelPricing_DeepseekUsesOfficialRatesForStaleEntries(t *testing.T) {
	pricingService := newCatalogFixture(catalogFixture{pricingData: map[string]*billingpricing.CatalogModelPricing{
		"deepseek-v4-pro":      {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 3e-8},
		"deepseek-v4-flash":    {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 3e-8},
		"deepseek-v3-2-251201": {InputCostPerToken: 0, OutputCostPerToken: 0},
	}})
	bs := newCalculator(&config.Config{}, pricingService)

	tests := []struct {
		model                 string
		input, output, cached float64
	}{
		{model: "deepseek-v4-pro", input: billingpricing.DeepseekProOffPeakInputPrice, output: billingpricing.DeepseekProOffPeakOutputPrice, cached: billingpricing.DeepseekProOffPeakCacheRead},
		{model: "deepseek-v4-flash", input: billingpricing.DeepseekFlashOffPeakInputPrice, output: billingpricing.DeepseekFlashOffPeakOutputPrice, cached: billingpricing.DeepseekFlashOffPeakCacheRead},
		{model: "deepseek-v4-pro-0813", input: billingpricing.DeepseekProOffPeakInputPrice, output: billingpricing.DeepseekProOffPeakOutputPrice, cached: billingpricing.DeepseekProOffPeakCacheRead},
		{model: "deepseek-v3-2-251201", input: 0, output: 0, cached: 0},
		{model: "deepseek-unknown", input: billingpricing.DeepseekFlashOffPeakInputPrice, output: billingpricing.DeepseekFlashOffPeakOutputPrice, cached: billingpricing.DeepseekFlashOffPeakCacheRead},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			// 价格入口使用 TokenFlux 当前统一计算器。
			pricing, err := bs.GetModelPricing(tt.model)
			if tt.model == "deepseek-v4-pro-0813" || tt.model == "deepseek-unknown" {
				require.ErrorIs(t, err, billingpricing.ErrModelPricingUnavailable)
				require.Nil(t, pricing)
				return
			}
			require.NoError(t, err)
			require.InDelta(t, tt.input, pricing.InputPricePerToken, 1e-15)
			require.InDelta(t, tt.output, pricing.OutputPricePerToken, 1e-15)
			require.InDelta(t, tt.cached, pricing.CacheReadPricePerToken, 1e-15)
		})
	}
}

func TestCalculateCostUnified_DeepseekPeakDoesNotOverrideConfigPricing(t *testing.T) {
	bs := newCalculator(&config.Config{}, nil)
	inputPrice, outputPrice := 1e-6, 2e-6
	resolver, _ := settingsResolver(bs, billingpricing.DefaultBillingSettings(), []routing.ModelPricingEntry{{
		Models:      []string{"deepseek-v4-flash"},
		BillingMode: routing.BillingModeToken,
		InputPrice:  &inputPrice,
		OutputPrice: &outputPrice,
	}})
	tokens := billingpricing.UsageTokens{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 1000}
	for _, pricingAt := range []time.Time{
		time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 24, 2, 0, 0, 0, time.UTC),
	} {
		cost, err := bs.CalculateCostUnified(billing.CostInput{
			Ctx: context.Background(), Model: "deepseek-v4-flash", GroupID: billingtestkit.GroupID(),
			Tokens: tokens, RateMultiplier: 1, PricingAt: pricingAt, Resolver: resolver,
		})
		require.NoError(t, err)
		require.InDelta(t, 1000*inputPrice+500*outputPrice+1000*billingpricing.DeepseekFlashOffPeakCacheRead, cost.TotalCost, 1e-12)
	}
}

// TestDeepseekDefaultCatalogUsesNativeEntries 验证默认目录原厂报价；中继的历史型号仍可独立存在。
func TestDeepseekDefaultCatalogUsesNativeEntries(t *testing.T) {
	pricingService := newOfflineCatalogFixture(t)
	pricingData := pricingService.Snapshot().Data

	for _, tc := range []struct {
		model                 string
		input, output, cached float64
	}{
		{"deepseek-v4-flash", 0.15e-6, 0.6e-6, 0.003e-6},
		{"deepseek-v4-flash-vision-exp", 0.15e-6, 0.6e-6, 0.003e-6},
		{"deepseek-v4-pro", 0.435e-6, 0.87e-6, 0.003625e-6},
	} {
		entry, exists := pricingData["deepseek/"+tc.model]
		require.True(t, exists, "%s 必须存在于原厂价格目录", tc.model)
		require.NotNil(t, entry)
		require.Equal(t, "models.dev", entry.Source)
		require.Equal(t, "deepseek", entry.Provider)
		require.InDelta(t, tc.input, entry.InputCostPerToken, 1e-15)
		require.InDelta(t, tc.output, entry.OutputCostPerToken, 1e-15)
		require.InDelta(t, tc.cached, entry.CacheReadInputTokenCost, 1e-15)
	}
}
