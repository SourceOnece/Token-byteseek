package pricing

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// 使用新目录解析入口验证规则，不重新引入按模型名硬编码的回退路径。
func testByteSeekCatalogPrice(t *testing.T, model, body string) *ModelPricing {
	t.Helper()
	prices, diagnostics, err := ParsePricingEntries(map[string]json.RawMessage{model: json.RawMessage(body)})
	require.NoError(t, err)
	require.NoError(t, diagnostics.ValidationError())
	price, err := ResolveModelPricing(model, prices[model])
	require.NoError(t, err)
	return price
}

// 缓存派生规则只补缺失值，目录显式零价仍然免费。
func TestGPT61MissingCacheWrite(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		fields := map[string]any{"input_cost_per_token": 2e-6, "output_cost_per_token": 10e-6, "cache_write_multiplier": 1.25}
		if explicit {
			fields["cache_creation_input_token_cost"] = 0
		}
		body, err := json.Marshal(fields)
		require.NoError(t, err)
		price := testByteSeekCatalogPrice(t, "gpt-6.1-sol", string(body))
		if explicit {
			require.Zero(t, price.CacheCreationPricePerToken)
		} else {
			require.InDelta(t, 2.5e-6, price.CacheCreationPricePerToken, 1e-12)
		}
	}
}

// Ultrafast 独立规则不能被 Fast 覆盖；标准和长上下文仍按同一目录值计算。
func TestGPT61AndAstraPricing(t *testing.T) {
	sol := testByteSeekCatalogPrice(t, "gpt-6.1-sol", `{"input_cost_per_token":0.000002,"output_cost_per_token":0.000010,"fast_multiplier":2,"long_context_input_token_threshold":272000,"long_context_input_cost_multiplier":2,"long_context_output_cost_multiplier":1.5}`)
	usage := UsageTokens{InputTokens: 1000, OutputTokens: 1000}
	require.InDelta(t, .012, ComputeTokenBreakdown(sol, usage, 1, "default", true).TotalCost, 1e-12)
	require.InDelta(t, .024, ComputeTokenBreakdown(sol, usage, 1, "priority", true).TotalCost, 1e-12)
	require.InDelta(t, 272001*4e-6, ComputeTokenBreakdown(sol, UsageTokens{InputTokens: 272001}, 1, "default", true).TotalCost, 1e-12)
	astra := testByteSeekCatalogPrice(t, "gpt-6-astra", `{"input_cost_per_token":0.000010,"output_cost_per_token":0.000050,"ultrafast_multiplier":6}`)
	fast := 3.0
	astra.FastModeMultiplier = &fast
	require.InDelta(t, .36, ComputeTokenBreakdown(astra, usage, 1, "ultrafast", true).TotalCost, 1e-12)
	require.InDelta(t, .18, ComputeTokenBreakdown(astra, usage, 1, "priority", true).TotalCost, 1e-12)
}
