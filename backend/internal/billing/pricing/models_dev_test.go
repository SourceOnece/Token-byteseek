package pricing

import (
	"encoding/json"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/modelcatalog"
	"github.com/stretchr/testify/require"
)

func TestModelsDevContextPricesAndCacheTTL(t *testing.T) {
	catalog, err := modelcatalog.Parse([]byte(`{"models":{"anthropic/claude-test":{"name":"Claude"}},"providers":{"anthropic":{"models":{"claude-test":{"cost":{"input":3,"output":15,"cache_write":3.75,"cache_read":0.3,"tiers":[{"tier":{"type":"context","size":100},"input":6,"output":30,"cache_write":7.5,"cache_read":0},{"tier":{"type":"context","size":200},"input":9,"output":45,"cache_write":11.25,"cache_read":0.9}]}}}}}}`))
	require.NoError(t, err)
	entries, _, err := ParsePricingEntries(ModelsDevPrices(catalog))
	require.NoError(t, err)
	base, _, err := ResolveModelPricing("claude-test", entries["claude-test"], nil, ModelPolicy{})
	require.NoError(t, err)
	for _, tc := range []struct {
		input int
		price float64
	}{{100, 3}, {101, 6}, {200, 6}, {201, 9}} {
		cost := ComputeTokenBreakdown(base, UsageTokens{InputTokens: tc.input}, 1, "", true)
		require.InDelta(t, float64(tc.input)*tc.price/1e6, cost.TotalCost, 1e-12)
	}
	cache := ComputeTokenBreakdown(base, UsageTokens{CacheCreationTokens: 30, CacheCreation5mTokens: 10, CacheCreation1hTokens: 20}, 1, "", true)
	require.InDelta(t, (10*3.75+20*6)/1e6, cache.TotalCost, 1e-12)
	intervals := LongContextDisplayPricingIntervals(base, 2)
	require.Len(t, intervals, 3)
	require.InDelta(t, 12e-6, intervals[1].InputPricePerToken, 1e-12)
	require.Equal(t, 0.0, intervals[1].CacheReadPricePerToken)
	require.Nil(t, WithoutLongContextDisplayPricing(base).ContextPrices)
	paid := MultiplyModelPricing(base, 2)
	require.InDelta(t, 101*12e-6, ComputeTokenBreakdown(paid, UsageTokens{InputTokens: 101}, 1, "", true).TotalCost, 1e-12)
}

func TestModelsDevExplicitFreeFastAndCacheTTL(t *testing.T) {
	catalog, err := modelcatalog.Parse([]byte(`{"models":{"anthropic/claude-test":{"name":"Claude"},"openai/gpt-5.5":{"name":"GPT"}},"providers":{"anthropic":{"models":{"claude-test":{"cost":{"input":3,"output":15,"cache_write":3.75},"experimental":{"modes":{"fast":{"cost":{"input":6,"output":30,"cache_write":7.5}}}}}}},"openai":{"models":{"gpt-5.5":{"cost":{"input":3,"output":15},"experimental":{"modes":{"fast":{"cost":{"input":0,"output":0}}}}}}}}}`))
	require.NoError(t, err)
	entries, _, err := ParsePricingEntries(ModelsDevPrices(catalog))
	require.NoError(t, err)
	claude, _, err := ResolveModelPricing("claude-test", entries["claude-test"], nil, ModelPolicy{})
	require.NoError(t, err)
	usage := UsageTokens{CacheCreationTokens: 30, CacheCreation5mTokens: 10, CacheCreation1hTokens: 20}
	cost := ComputeTokenBreakdown(claude, usage, 1, "priority", true)
	require.InDelta(t, (10*7.5+20*12)/1e6, cost.TotalCost, 1e-12)
	fast, ok := FastModeDisplayPricing(claude)
	require.True(t, ok)
	require.InDelta(t, 12e-6, fast.CacheCreation1hPrice, 1e-12)
	gpt, _, err := ResolveModelPricing("gpt-5.5", entries["gpt-5.5"], nil, ModelPolicy{NormalizedOpenAIModel: "gpt-5.5"})
	require.NoError(t, err)
	require.Zero(t, ComputeTokenBreakdown(gpt, UsageTokens{InputTokens: 10, OutputTokens: 10}, 1, "priority", true).TotalCost)
}

func TestModelsDevExplicitZeroOneHourCachePrice(t *testing.T) {
	entries, _, err := ParsePricingEntries(map[string]json.RawMessage{"claude-test": json.RawMessage(`{"source":"models.dev","input_cost_per_token":0.000003,"output_cost_per_token":0.000015,"cache_creation_input_token_cost":0.00000375,"cache_creation_input_token_cost_above_1hr":0}`)})
	require.NoError(t, err)
	base, _, err := ResolveModelPricing("claude-test", entries["claude-test"], nil, ModelPolicy{})
	require.NoError(t, err)
	require.True(t, base.SupportsCacheBreakdown)
	require.Zero(t, ComputeTokenBreakdown(base, UsageTokens{CacheCreationTokens: 10, CacheCreation1hTokens: 10}, 1, "", true).TotalCost)
}

func TestModelsDevZeroCacheWriteDoesNotUseModelFallback(t *testing.T) {
	entries, _, err := ParsePricingEntries(map[string]json.RawMessage{"gpt-5.6-sol": json.RawMessage(`{"source":"models.dev","input_cost_per_token":0.000003,"output_cost_per_token":0.000015,"cache_creation_input_token_cost":0}`)})
	require.NoError(t, err)
	base, _, err := ResolveModelPricing("gpt-5.6-sol", entries["gpt-5.6-sol"], nil, ModelPolicy{IsGPT56: true, NormalizedOpenAIModel: "gpt-5.6-sol"})
	require.NoError(t, err)
	require.Zero(t, ComputeTokenBreakdown(base, UsageTokens{CacheCreationTokens: 10}, 1, "", true).TotalCost)
}

// TestModelsDevImageOutputScope 限定生图费率解释范围，避免根据图片模态改写研究模型或中继报价。
func TestModelsDevImageOutputScope(t *testing.T) {
	catalog, err := modelcatalog.Parse([]byte(`{"providers":{
		"google":{"models":{
			"gemini-image-test":{"cost":{"input":2,"output":120},"modalities":{"output":["text","image"]}},
			"deep-research-test":{"cost":{"input":2,"output":12},"modalities":{"output":["text","image"]}},
			"gemini-text-test":{"cost":{"input":2,"output":12},"modalities":{"output":["text"]}},
			"gemini-omni-test":{"cost":{"input":2,"output":12},"modalities":{"output":["text","image","video"]}}
		}},
		"openrouter":{"models":{"gemini-image-test":{"cost":{"input":1,"output":9},"modalities":{"output":["text","image"]}}}}
	}}`))
	require.NoError(t, err)
	entries, _, err := ParsePricingEntries(ModelsDevPrices(catalog))
	require.NoError(t, err)
	for model, price := range map[string]float64{"deep-research-test": 12e-6, "gemini-text-test": 12e-6, "gemini-omni-test": 12e-6, "openrouter/gemini-image-test": 9e-6} {
		value, _, err := ResolveModelPricing(model, entries[model], nil, ModelPolicy{})
		require.NoError(t, err)
		require.InDelta(t, price, value.OutputPricePerToken, 1e-12)
	}
	// 已知缺少文本价时，不借用跨模型或旧媒体兜底价。
	fallback := map[string]*ModelPricing{"gemini-image-test": {InputPricePerToken: 2e-6, OutputPricePerToken: 99e-6}}
	_, _, err = ResolveModelPricing("gemini-image-test", entries["gemini-image-test"], fallback, ModelPolicy{})
	require.ErrorIs(t, err, ErrModelPricingUnavailable)
}
