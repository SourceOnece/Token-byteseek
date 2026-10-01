package pricing

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPriorityCacheTTLUsesPricesRegardlessOfSource 验证来源只作追溯，不改变 Fast TTL 金额和展示。
func TestPriorityCacheTTLUsesPricesRegardlessOfSource(t *testing.T) {
	for _, source := range []string{"models.dev", "local_supplement"} {
		t.Run(source, func(t *testing.T) {
			entries, diagnostics, err := ParsePricingEntries(map[string]json.RawMessage{"custom": json.RawMessage(`{"source":"` + source + `","input_cost_per_token":0.000001,"output_cost_per_token":0.000002,"input_cost_per_token_priority":0.000002,"output_cost_per_token_priority":0.000004,"cache_write_multiplier":1.25,"cache_write_1h_multiplier":2}`)})
			require.NoError(t, err)
			require.NoError(t, diagnostics.ValidationError())
			price, err := ResolveModelPricing("custom", entries["custom"])
			require.NoError(t, err)
			for _, tc := range []struct {
				tokens UsageTokens
				cost   float64
			}{
				{UsageTokens{CacheCreationTokens: 100, CacheCreation5mTokens: 100}, 0.00025},
				{UsageTokens{CacheCreationTokens: 100, CacheCreation1hTokens: 100}, 0.0004},
				{UsageTokens{CacheCreationTokens: 200, CacheCreation5mTokens: 100, CacheCreation1hTokens: 100}, 0.00065},
			} {
				cost := ComputeTokenBreakdown(price, tc.tokens, 1, "priority", true)
				require.InDelta(t, tc.cost, cost.TotalCost, 1e-12)
			}
			fast, ok := FastModeDisplayPricing(price)
			require.True(t, ok)
			require.InDelta(t, 2.5e-6, fast.CacheCreation5mPrice, 1e-12)
			require.InDelta(t, 4e-6, fast.CacheCreation1hPrice, 1e-12)
			require.InDelta(t, 1.25e-6, price.CacheCreation5mPrice, 1e-12, "不得修改标准档")
		})
	}
}

// TestPartialImageCardDoesNotDeclareMissingSizesFree 覆盖纯价卡投影，目录补全由应用层统一完成。
func TestPartialImageCardDoesNotDeclareMissingSizesFree(t *testing.T) {
	zero := 0.0
	resolved := ResolvePriceCards(&ModelPricingEntry{BillingMode: BillingModeImage, Intervals: []PricingInterval{{TierLabel: "1K", PerRequestPrice: &zero}}}, nil, PricingSourceUnpriced, true)
	shown, ok := DisplayPricingFromResolved("custom-image", 1, resolved)
	require.True(t, ok)
	require.Equal(t, []string{"1K"}, shown.ImagePriceSizes)
	require.Zero(t, shown.ImagePrice1K)
}
