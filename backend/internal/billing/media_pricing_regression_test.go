package billing

import (
	"context"
	"fmt"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/stretchr/testify/require"
)

// mediaPriceCards 为回归用例提供完整型号的显式价格配置。
type mediaPriceCards struct {
	card *ModelPricingEntry
}

// GetEffectiveConfigModelPricing 返回测试显式配置的价卡。
func (s mediaPriceCards) GetEffectiveConfigModelPricing(context.Context, int64, string) *ModelPricingEntry {
	return s.card
}

// TestMediaUnitPriceRejectsMissingPrice 验证缺少独立按张报价时不能生成通用单价。
func TestMediaUnitPriceRejectsMissingPrice(t *testing.T) {
	calculator := NewCalculator(defaultCatalogStub{entries: map[string]*pricing.CatalogModelPricing{
		"grok-imagine-video-1.5": {CatalogRules: pricing.CatalogRules{VideoPrices: map[string]float64{"480p": 0.08}}, TokenPricingAbsent: true},
		"token-image":            {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6},
	}}, CalculatorOptions{})
	resolver := NewPriceResolver(nil, calculator, nil, nil)
	for _, model := range []string{"gemini-3.8-flash-image-high", "grok-imagine", "token-image", "grok-imagine-video"} {
		_, err := resolver.ResolveImageUnitPrice(context.Background(), PricingInput{Model: model}, "1K")
		require.ErrorIs(t, err, pricing.ErrModelPricingUnavailable, model)
		cost, err := calculator.CalculateImageCost(model, "1K", 1, 1)
		require.ErrorIs(t, err, pricing.ErrModelPricingUnavailable, model)
		require.Nil(t, cost)
	}
	for _, model := range []string{"grok-imagine-video-1.5-preview", "custom-video", "grok-imagine-image-quality"} {
		cost, err := calculator.CalculateVideoCost(model, "480p", 1, 5, 1)
		require.ErrorIs(t, err, pricing.ErrModelPricingUnavailable, model)
		require.Nil(t, cost)
	}
	video, err := calculator.CalculateVideoCost("grok-imagine-video-1.5", "480p", 1, 5, 1)
	require.NoError(t, err)
	require.InDelta(t, 0.4, video.ActualCost, 1e-12)
}

// TestPublicQuotePreservesExactImagePrices 验证仅有按张报价的型号仍显示价格，零价不等于缺价。
func TestPublicQuotePreservesExactImagePrices(t *testing.T) {
	calculator := NewCalculator(defaultCatalogStub{entries: map[string]*pricing.CatalogModelPricing{
		"grok-imagine-image-quality": {CatalogRules: pricing.CatalogRules{ImagePrices: map[string]float64{"1K": 0.05}}, TokenPricingAbsent: true},
		"custom-image":               {OutputCostPerImage: 0.1, ImagePricePresent: true, TokenPricingAbsent: true, Mode: "image_generation"},
		"free-image":                 {ImagePricePresent: true, TokenPricingAbsent: true, Mode: "image_generation"},
	}}, CalculatorOptions{})
	resolver := NewPriceResolver(nil, calculator, nil, nil)
	for _, tc := range []struct {
		model string
		price float64
	}{
		{"grok-imagine-image-quality", 0.05},
		{"custom-image", 0.1},
		{"free-image", 0},
	} {
		input := PricingInput{Model: tc.model}
		quote := resolver.PublicQuote(context.Background(), PublicQuoteInput{PricingInput: input, RateMultiplier: 2})
		require.Equal(t, "priced", quote.PriceStatus, tc.model)
		require.Equal(t, "image", quote.PricingMode, tc.model)
		require.InDelta(t, tc.price*2, quote.ImagePrice1K, 1e-12)
		unit, err := resolver.ResolveImageUnitPrice(context.Background(), input, "1K")
		require.NoError(t, err)
		require.Equal(t, tc.price, unit)
		cost, err := calculator.CalculateImageCost(tc.model, "1K", 1, 2)
		require.NoError(t, err)
		require.Equal(t, quote.ImagePrice1K, cost.ActualCost)
	}
	quote := resolver.PublicQuote(context.Background(), PublicQuoteInput{PricingInput: PricingInput{Model: "unknown-image"}, RateMultiplier: 1})
	require.Equal(t, "unpriced", quote.PriceStatus)
}

// TestMediaExplicitFreeCardWins 验证完整型号显式零价优先于目录和静态价格。
func TestMediaExplicitFreeCardWins(t *testing.T) {
	zero := 0.0
	groupID := int64(1)
	resolver := NewPriceResolver(mediaPriceCards{card: &ModelPricingEntry{
		BillingMode: pricing.BillingModeImage, PerRequestPrice: &zero,
	}}, NewCalculator(nil, CalculatorOptions{}), nil, nil)
	input := PricingInput{Model: "custom-image", GroupID: &groupID}
	unit, err := resolver.ResolveImageUnitPrice(context.Background(), input, "1K")
	require.NoError(t, err)
	require.Zero(t, unit)
	quote := resolver.PublicQuote(context.Background(), PublicQuoteInput{PricingInput: input, RateMultiplier: 2})
	require.Equal(t, "priced", quote.PriceStatus)
	require.Zero(t, quote.ImagePrice1K)
}

// TestPricingManagementFreeImageOverridesCatalog 验证价格管理零价优先于目录尺寸价，且不会修改目录成本。
func TestPricingManagementFreeImageOverridesCatalog(t *testing.T) {
	zero := 0.0
	groupID := int64(1)
	catalog := defaultCatalogStub{entries: map[string]*pricing.CatalogModelPricing{
		"gemini-3-pro-image": {CatalogRules: pricing.CatalogRules{ImagePrices: pricing.MediaPrices{"1K": 0.134, "2K": 0.134, "4K": 0.24}}, TokenPricingAbsent: true},
	}}
	calculator := NewCalculator(catalog, CalculatorOptions{})
	resolver := NewPriceResolver(mediaPriceCards{card: &ModelPricingEntry{BillingMode: pricing.BillingModeImage, PerRequestPrice: &zero}}, calculator, nil, nil)
	input := PricingInput{Model: "gemini-3-pro-image", GroupID: &groupID}
	shown := resolver.PublicQuote(context.Background(), PublicQuoteInput{PricingInput: input, RateMultiplier: 2})
	require.Equal(t, "priced", shown.PriceStatus)
	require.ElementsMatch(t, []string{"1K", "2K", "4K"}, shown.ImagePriceSizes)
	for _, size := range shown.ImagePriceSizes {
		unit, err := resolver.ResolveImageUnitPrice(context.Background(), input, size)
		require.NoError(t, err)
		require.Zero(t, unit)
		official, err := calculator.DefaultImagePrice(input.Model, size)
		require.NoError(t, err)
		require.Positive(t, official, "用户售价不能覆盖目录及提供商成本的基础数据")
	}
}

// TestPartialImageCardQuoteMatchesReservation 按实际查价结果标记尺寸，不将空白尺寸视作免费。
func TestPartialImageCardQuoteMatchesReservation(t *testing.T) {
	for _, oneK := range []float64{0, 0.1} {
		t.Run(fmt.Sprint(oneK), func(t *testing.T) {
			groupID := int64(1)
			calculator := NewCalculator(defaultCatalogStub{entries: map[string]*pricing.CatalogModelPricing{
				"custom-image": {CatalogRules: pricing.CatalogRules{ImagePrices: pricing.MediaPrices{"1K": 0.5, "2K": 0.134}}, TokenPricingAbsent: true},
			}}, CalculatorOptions{})
			resolver := NewPriceResolver(mediaPriceCards{card: &ModelPricingEntry{BillingMode: pricing.BillingModeImage, Intervals: []pricing.PricingInterval{{TierLabel: "1K", PerRequestPrice: &oneK}}}}, calculator, nil, nil)
			input := PricingInput{Model: "custom-image", GroupID: &groupID}
			quote := resolver.PublicQuote(context.Background(), PublicQuoteInput{PricingInput: input, RateMultiplier: 2})
			require.Equal(t, "priced", quote.PriceStatus)
			require.ElementsMatch(t, []string{"1K", "2K"}, quote.ImagePriceSizes)
			require.InDelta(t, oneK*2, quote.ImagePrice1K, 1e-12)
			unit, err := resolver.ResolveImageUnitPrice(context.Background(), input, "2K")
			require.NoError(t, err)
			require.InDelta(t, unit*2, quote.ImagePrice2K, 1e-12)
			cost, err := calculator.CalculateCostUnified(CostInput{Ctx: context.Background(), Model: input.Model, GroupID: input.GroupID, RequestCount: 3, SizeTier: "2K", RateMultiplier: 2, Resolver: resolver})
			require.NoError(t, err)
			require.InDelta(t, quote.ImagePrice2K*3, cost.ActualCost, 1e-12)
			_, err = resolver.ResolveImageUnitPrice(context.Background(), input, "4K")
			require.ErrorIs(t, err, pricing.ErrModelPricingUnavailable)
		})
	}
}
