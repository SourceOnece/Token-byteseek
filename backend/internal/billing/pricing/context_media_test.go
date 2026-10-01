package pricing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestContextTierImageOverrides 保证价卡图片价格在多档结算与展示中一致，并保留默认目录。
func TestContextTierImageOverrides(t *testing.T) {
	input, imageInput, imageOutput, zero, multiplier := 2e-6, 1e-6, 3e-6, 0.0, 2.0
	base := &ModelPricing{
		InputPricePerToken:       2e-6,
		OutputPricePerToken:      5e-6,
		ImageInputPricePerToken:  9e-6,
		ImageOutputPricePerToken: 10e-6,
		ContextPrices: []ContextModelPrice{
			{
				Threshold: 100,
				Pricing: &ModelPricing{
					InputPricePerToken:       4e-6,
					OutputPricePerToken:      10e-6,
					ImageInputPricePerToken:  18e-6,
					ImageOutputPricePerToken: 20e-6,
				},
			},
			{
				Threshold: 200,
				Pricing: &ModelPricing{
					InputPricePerToken:       8e-6,
					OutputPricePerToken:      20e-6,
					ImageInputPricePerToken:  36e-6,
					ImageOutputPricePerToken: 40e-6,
				},
			},
		},
	}
	before := MultiplyModelPricing(base, 1)
	for _, tc := range []struct {
		name                    string
		config                  ModelPricingEntry
		imageInput, imageOutput float64
	}{
		{"custom", ModelPricingEntry{ImageInputPrice: &imageInput, ImageOutputPrice: &imageOutput}, 1e-6, 3e-6},
		{"scaled", ModelPricingEntry{ImageInputPrice: &imageInput, ImageOutputPrice: &imageOutput, PriceMultiplier: &multiplier}, 2e-6, 6e-6},
		{"zero", ModelPricingEntry{InputPrice: &input, ImageInputPrice: &zero, ImageOutputPrice: &zero}, 0, 0},
		{"omitted", ModelPricingEntry{InputPrice: &input}, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolved := ResolvePriceCards(&tc.config, base, PricingSourceCatalog, true)
			for _, tokens := range []int{100, 101, 200, 201} {
				cost, err := CalculateTokenCost(resolved, CostInput{
					Model: "test",
					Tokens: UsageTokens{
						InputTokens:       tokens,
						ImageInputTokens:  10,
						OutputTokens:      10,
						ImageOutputTokens: 10,
					},
					RateMultiplier: 1,
				})
				require.NoError(t, err)
				inputPrice := tc.imageInput
				if inputPrice == 0 {
					inputPrice = input
				}
				require.InDelta(t, 10*inputPrice, cost.ImageInputCost, 1e-12)
				require.InDelta(t, 10*tc.imageOutput, cost.ImageOutputCost, 1e-12)
			}
			for _, value := range []*ModelPricing{resolved.BasePricing, ApplyConfigPrice(base, &tc.config)} {
				intervals := LongContextDisplayPricingIntervals(value, 1)
				require.Len(t, intervals, 3)
				for _, interval := range intervals {
					require.InDelta(t, tc.imageInput, interval.ImageInputPricePerToken, 1e-12)
					require.InDelta(t, tc.imageOutput, interval.ImageOutputPricePerToken, 1e-12)
				}
				for _, tier := range value.ContextPrices {
					require.True(t, tier.Pricing.ImageOutputPriceExplicit)
				}
			}
			require.Equal(t, before, MultiplyModelPricing(base, 1))
		})
	}
}
