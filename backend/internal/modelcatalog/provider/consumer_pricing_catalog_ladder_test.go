package provider

import (
	"testing"

	billingpricing "github.com/TokenFlux/TokenRouter/internal/billing/pricing"

	"github.com/stretchr/testify/require"
)

func TestParsePricingData_DerivesLongContextFromAboveTierFields(t *testing.T) {
	data, err := parsePricingFixture([]byte(`{
		"gpt-above": {"provider": "openai", "mode": "chat",
			"input_cost_per_token": 5e-06, "output_cost_per_token": 3e-05,
			"input_cost_per_token_above_272k_tokens": 1e-05,
			"output_cost_per_token_above_272k_tokens": 4.5e-05,
			"input_cost_per_token_above_272k_tokens_flex": 5e-06},
		"gemini-above": {"provider": "vertex_ai-language-models", "mode": "chat",
			"input_cost_per_token": 1.25e-06, "output_cost_per_token": 1e-05,
			"input_cost_per_token_above_200k_tokens": 2.5e-06,
			"output_cost_per_token_above_200k_tokens": 1.5e-05},
		"explicit-wins": {"provider": "openai", "mode": "chat",
			"input_cost_per_token": 5e-06, "output_cost_per_token": 3e-05,
			"long_context_input_cost_multiplier": 1,
			"input_cost_per_token_above_272k_tokens": 1e-05,
			"output_cost_per_token_above_272k_tokens": 4.5e-05},
		"no-surcharge": {"provider": "openai", "mode": "chat",
			"input_cost_per_token": 5e-06, "output_cost_per_token": 3e-05,
			"input_cost_per_token_above_272k_tokens": 5e-06,
			"output_cost_per_token_above_272k_tokens": 3e-05},
		"multi-threshold": {"provider": "openai", "mode": "chat",
			"input_cost_per_token": 1e-06, "output_cost_per_token": 2e-06,
			"input_cost_per_token_above_128k_tokens": 2e-06,
			"input_cost_per_token_above_272k_tokens": 4e-06}
	}`))
	require.NoError(t, err)

	require.Equal(t, 272000, data["gpt-above"].LongContextInputTokenThreshold)
	require.InDelta(t, 2.0, data["gpt-above"].LongContextInputCostMultiplier, 1e-12)
	require.InDelta(t, 1.5, data["gpt-above"].LongContextOutputCostMultiplier, 1e-12)
	require.Equal(t, 200000, data["gemini-above"].LongContextInputTokenThreshold)
	require.Zero(t, data["explicit-wins"].LongContextInputTokenThreshold)
	require.Zero(t, data["no-surcharge"].LongContextInputTokenThreshold)
	require.Equal(t, 128000, data["multi-threshold"].LongContextInputTokenThreshold)
}

func TestGetModelPricing_XAIThresholdInclusive(t *testing.T) {
	service := newBillingFixture(newStubCatalogFromJSON(t, `{
		"grok-4.5": {"provider": "xai", "mode": "chat",
			"input_cost_per_token": 2e-06, "output_cost_per_token": 6e-06,
			"input_cost_per_token_above_200k_tokens": 4e-06,
			"output_cost_per_token_above_200k_tokens": 1.2e-05}
	}`))
	pricing, err := service.GetModelPricing("grok-4.5")
	require.NoError(t, err)
	require.Equal(t, 200000, pricing.LongContextInputThreshold)
	require.True(t, pricing.LongContextThresholdInclusive)
}

func TestParsePricingData_ExplicitZeroThresholdDisablesLadder(t *testing.T) {
	data, err := parsePricingFixture([]byte(`{
		"gpt-5.5": {"provider": "openai", "mode": "chat",
			"input_cost_per_token": 5e-06, "output_cost_per_token": 3e-05,
			"long_context_input_token_threshold": 0,
			"input_cost_per_token_above_272k_tokens": 1e-05,
			"output_cost_per_token_above_272k_tokens": 4.5e-05}
	}`))
	require.NoError(t, err)
	require.Zero(t, data["gpt-5.5"].LongContextInputTokenThreshold)
	require.Zero(t, data["gpt-5.5"].LongContextInputCostMultiplier)
}

func TestParsePricingData_WarnsOrphanCacheTierFields(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	service := newHotReloadCatalog(t, `{
		"gemini-orphan": {"provider": "vertex_ai-language-models", "mode": "chat",
			"input_cost_per_token": 1.25e-06, "output_cost_per_token": 1e-05,
			"input_cost_per_token_above_200k_tokens": 2.5e-06,
			"output_cost_per_token_above_200k_tokens": 1.5e-05,
			"cache_creation_input_token_cost_above_200k_tokens": 2.5e-07},
		"gemini-complete": {"provider": "vertex_ai-language-models", "mode": "chat",
			"input_cost_per_token": 1.25e-06, "output_cost_per_token": 1e-05,
			"cache_creation_input_token_cost": 1.25e-06,
			"input_cost_per_token_above_200k_tokens": 2.5e-06,
			"output_cost_per_token_above_200k_tokens": 1.5e-05,
			"cache_creation_input_token_cost_above_200k_tokens": 2.5e-06},
		"priority-orphan": {"provider": "openai", "mode": "chat",
			"input_cost_per_token": 5e-06, "output_cost_per_token": 3e-05,
			"cache_creation_input_token_cost_above_272k_tokens_priority": 2.5e-05}
	}`)
	data := service.Snapshot().Data
	require.Equal(t, 200000, data["gemini-orphan"].LongContextInputTokenThreshold)
	require.True(t, logSink.ContainsMessageAtLevel("gemini-orphan(cache_creation_input_token_cost_above_200k_tokens)", "warn"))
	require.True(t, logSink.ContainsMessage("priority-orphan(cache_creation_input_token_cost_above_272k_tokens_priority)"))
	require.False(t, logSink.ContainsMessage("gemini-complete"))
}

func TestParsePricingData_WarnsLopsidedLongContextLadder(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	service := newHotReloadCatalog(t, `{
		"mixed-versions": {"provider": "openai", "mode": "chat",
			"input_cost_per_token": 5e-06, "output_cost_per_token": 3e-05,
			"input_cost_per_token_above_272k_tokens": 8e-06,
			"output_cost_per_token_above_272k_tokens": 3e-05},
		"consistent": {"provider": "openai", "mode": "chat",
			"input_cost_per_token": 4e-06, "output_cost_per_token": 2e-05,
			"input_cost_per_token_above_272k_tokens": 8e-06,
			"output_cost_per_token_above_272k_tokens": 3e-05}
	}`)
	data := service.Snapshot().Data
	require.Equal(t, 272000, data["mixed-versions"].LongContextInputTokenThreshold)
	require.True(t, logSink.ContainsMessageAtLevel("mixed-versions(input x1.60, output x1.00)", "warn"))
	require.False(t, logSink.ContainsMessage("consistent"))
}

// TestDefaultCatalogSnapshot_CacheTierContract 验证发布目录的真实阶梯与缓存报价。
func TestDefaultCatalogSnapshot_CacheTierContract(t *testing.T) {
	service := newOfflinePricingFixture(t)
	price := service.GetModelPricing("gpt-5.6-sol")
	require.NotNil(t, price)
	require.Len(t, price.ContextPrices, 1)
	require.Equal(t, 272000, price.ContextPrices[0].Threshold)
	require.InDelta(t, 0.4e-6, price.CacheReadInputTokenCost, 1e-12)
	require.InDelta(t, 0.8e-6, price.ContextPrices[0].Pricing.CacheReadInputTokenCost, 1e-12)
}

func TestCalculateCost_PartialLongContextMultiplierDefaultsToOne(t *testing.T) {
	tokens := billingpricing.UsageTokens{InputTokens: 300000, OutputTokens: 1000, CacheReadTokens: 10000}

	t.Run("only input multiplier", func(t *testing.T) {
		service := newBillingFixture(newStubCatalogFromJSON(t, `{
			"partial-in": {"provider": "openai", "mode": "chat",
				"input_cost_per_token": 2e-06, "output_cost_per_token": 1e-05,
				"cache_read_input_token_cost": 2e-07,
				"long_context_input_token_threshold": 272000,
				"long_context_input_cost_multiplier": 2.0}
		}`))
		cost, err := service.CalculateCost("partial-in", tokens, 1)
		require.NoError(t, err)
		require.InDelta(t, 300000*2e-6*2, cost.InputCost, 1e-10)
		require.InDelta(t, 1000*1e-5, cost.OutputCost, 1e-10)
		require.InDelta(t, 10000*2e-7*2, cost.CacheReadCost, 1e-10)
	})

	t.Run("only output multiplier", func(t *testing.T) {
		service := newBillingFixture(newStubCatalogFromJSON(t, `{
			"partial-out": {"provider": "openai", "mode": "chat",
				"input_cost_per_token": 2e-06, "output_cost_per_token": 1e-05,
				"cache_read_input_token_cost": 2e-07,
				"long_context_input_token_threshold": 272000,
				"long_context_output_cost_multiplier": 1.5}
		}`))
		cost, err := service.CalculateCost("partial-out", tokens, 1)
		require.NoError(t, err)
		require.InDelta(t, 300000*2e-6, cost.InputCost, 1e-10)
		require.InDelta(t, 1000*1e-5*1.5, cost.OutputCost, 1e-10)
		require.InDelta(t, 10000*2e-7, cost.CacheReadCost, 1e-10)
	})
}

// 模型广场展示必须与结算路径使用相同的缺省倍率，避免部分覆盖把一侧显示成免费。
func TestDisplayPricing_PartialLongContextMultiplierDefaultsToOne(t *testing.T) {
	service := newBillingFixture(newStubCatalogFromJSON(t, `{
		"partial-display": {"provider": "openai", "mode": "chat",
			"input_cost_per_token": 2e-06, "output_cost_per_token": 1e-05,
			"long_context_input_token_threshold": 272000,
			"long_context_input_cost_multiplier": 2}
	}`))
	display := service.DisplayPricing("partial-display", 1)
	require.Len(t, display.ContextIntervals, 2)
	require.InDelta(t, 4e-6, display.ContextIntervals[1].InputPricePerToken, 1e-12)
	require.InDelta(t, 1e-5, display.ContextIntervals[1].OutputPricePerToken, 1e-12)
}

func TestCalculateCost_ClaudeSonnetCatalogLadderIsDataDriven(t *testing.T) {
	service := newBillingFixture(newStubCatalogFromJSON(t, `{
		"claude-sonnet-4-5": {"provider": "anthropic", "mode": "chat",
			"input_cost_per_token": 3e-06, "output_cost_per_token": 1.5e-05,
			"cache_read_input_token_cost": 3e-07,
			"input_cost_per_token_above_200k_tokens": 6e-06,
			"output_cost_per_token_above_200k_tokens": 2.25e-05}
	}`))

	pricing, err := service.GetModelPricing("claude-sonnet-4-5")
	require.NoError(t, err)
	require.Equal(t, 200000, pricing.LongContextInputThreshold)
	require.False(t, pricing.LongContextThresholdInclusive)
	cost, err := service.CalculateCost("claude-sonnet-4-5", billingpricing.UsageTokens{InputTokens: 250000, OutputTokens: 1000}, 1)
	require.NoError(t, err)
	require.True(t, cost.LongContextBillingApplied)
	require.InDelta(t, 250000*3e-6*2, cost.InputCost, 1e-10)
	require.InDelta(t, 1000*1.5e-5*1.5, cost.OutputCost, 1e-10)
}
