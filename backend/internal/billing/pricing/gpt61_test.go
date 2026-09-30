package pricing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 新模型独立报价，Astra 的 Ultrafast 与 Fast 配置互不覆盖。
func TestGPT61AndAstraPricing(t *testing.T) {
	prices := DefaultFallbackPrices()
	sol, _, err := ResolveModelPricing("gpt-6.1-sol", nil, prices, ModelPolicy{NormalizedOpenAIModel: "gpt-6.1-sol"})
	require.NoError(t, err)
	usage := UsageTokens{InputTokens: 1000, OutputTokens: 1000}
	require.InDelta(t, .012, ComputeTokenBreakdown(sol, usage, 1, "default", true).TotalCost, 1e-12)
	require.InDelta(t, .024, ComputeTokenBreakdown(sol, usage, 1, "priority", true).TotalCost, 1e-12)
	long := ComputeTokenBreakdown(sol, UsageTokens{InputTokens: 272001}, 1, "default", true)
	require.InDelta(t, 272001*4e-6, long.TotalCost, 1e-12)
	astra := ApplyModelSpecificPricingPolicy("gpt-6-astra", prices["gpt-6-astra"], ModelPolicy{NormalizedOpenAIModel: "gpt-6-astra"})
	fast := 3.0
	astra.FastModeMultiplier = &fast
	require.InDelta(t, .36, ComputeTokenBreakdown(astra, usage, 1, "ultrafast", true).TotalCost, 1e-12)
	require.InDelta(t, .18, ComputeTokenBreakdown(astra, usage, 1, "priority", true).TotalCost, 1e-12)
}
