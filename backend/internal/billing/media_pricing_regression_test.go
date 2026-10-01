package billing

import (
	"context"
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
		"token-image": {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6},
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
		"custom-image": {OutputCostPerImage: 0.1, ImagePricePresent: true, TokenPricingAbsent: true, Mode: "image_generation"},
		"free-image":   {ImagePricePresent: true, TokenPricingAbsent: true, Mode: "image_generation"},
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
