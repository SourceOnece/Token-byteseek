package provider

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/TokenFlux/TokenRouter/internal/modelcatalog"
	"github.com/stretchr/testify/require"
)

// TestSupplementRulesPublishAtomically 覆盖目录优先、独立单位、操作默认价及失败保留。
func TestSupplementRulesPublishAtomically(t *testing.T) {
	dir := t.TempDir()
	supplement := filepath.Join(dir, "supplement.json")
	body := `{"claude-test":{"input_cost_per_token":99,"cache_write_1h_multiplier":2,"fast_multiplier":3,"flex_multiplier":0.25,"max_reasoning_effort_multiplier":4,"image_prices":{"1K":0,"2K":0.7},"video_prices":{"720p":0.2},"time_pricing":{"timezone":"UTC","periods":[{"start_time":"01:00","end_time":"03:00","multiplier":2}]}},"_billing_defaults":{"web_search_price_per_call":0.25,"audio_tts_price_per_million_chars":10}}`
	require.NoError(t, os.WriteFile(supplement, []byte(body), 0o600))
	remote := &catalogRemoteFixture{body: []byte(modelsCatalogFixture), etag: "v1"}
	service := NewService(Options{DataDir: dir, RemoteURL: "https://models.dev/catalog.json", FallbackFile: supplement}, remote)
	require.NoError(t, service.ForceUpdate())
	raw := service.GetModelPricing("claude-test")
	require.InDelta(t, 3e-6, raw.InputCostPerToken, 1e-12)
	require.InDelta(t, 6e-6, raw.CacheCreationInputTokenCostAbove1hr, 1e-12)
	require.InDelta(t, 12e-6, raw.ContextPrices[0].Pricing.CacheCreationInputTokenCostAbove1hr, 1e-12)
	require.Equal(t, "rule_supplement", raw.PriceSources["cache_write_1h"])
	require.Equal(t, "local_supplement", raw.PriceSources["image_prices"])
	require.Nil(t, service.GetModelPricing(pricing.BillingDefaultsKey))
	require.NotContains(t, service.Snapshot().Data, pricing.BillingDefaultsKey)
	require.NotContains(t, service.ListModelNamesByProvider(""), pricing.BillingDefaultsKey)
	require.Equal(t, service.GetModelPricing("anthropic/claude-test").ImagePrices, raw.ImagePrices)
	calc := billing.NewCalculator(service, billing.CalculatorOptions{Now: func() time.Time { return time.Date(2026, 10, 1, 2, 0, 0, 0, time.UTC) }})
	unit, err := calc.DefaultImagePrice("claude-test", "1K")
	require.NoError(t, err)
	require.Zero(t, unit)
	_, err = calc.DefaultImagePrice("claude-test", "4K")
	require.ErrorIs(t, err, pricing.ErrModelPricingUnavailable)
	unit, err = calc.DefaultVideoPrice("claude-test", "720p")
	require.NoError(t, err)
	require.Equal(t, 0.2, unit)
	quote := calc.DefaultModelPrice("claude-test", "anthropic", "video")
	require.Equal(t, "priced", quote.PriceStatus)
	require.Len(t, quote.Prices, 1)
	require.Equal(t, "720p", quote.Prices[0].Key)
	require.Equal(t, 0.5, calc.CalculateWebSearchCost(2, nil, 1).ActualCost)
	zero := 0.0
	require.Zero(t, calc.CalculateWebSearchCost(2, &zero, 1).ActualCost)
	// 无长上下文和显式价卡时，时段与推理倍率由同一目录数据决定。
	resolver := billing.NewPriceResolver(nil, calc, nil, nil)
	cost, err := calc.CalculateCostUnified(billing.CostInput{Ctx: context.Background(), Model: "claude-test", Tokens: pricing.UsageTokens{InputTokens: 10}, RateMultiplier: 1, ServiceTier: "priority", ReasoningEffort: "max", Resolver: resolver})
	require.NoError(t, err)
	require.InDelta(t, 10*3e-6*3*4*2, cost.ActualCost, 1e-12)
	before := service.Snapshot()
	require.NoError(t, os.WriteFile(supplement, []byte(`{"claude-test":{"image_prices":{"1K":-1}},"_billing_defaults":{"web_search_price_per_call":2}}`), 0o600))
	require.Error(t, service.ForceUpdate())
	require.Equal(t, before.Data, service.Snapshot().Data)
	require.Equal(t, before.BillingDefaults, service.BillingDefaults())
	require.Equal(t, before.LastUpdated, service.Snapshot().LastUpdated)
	// 304 时仍重读补充文件，显式零价和删除字段同时生效。
	require.NoError(t, os.WriteFile(supplement, []byte(`{"claude-test":{"cache_creation_input_token_cost_above_1hr":0,"image_prices":{"1K":0}},"_billing_defaults":{"audio_tts_price_per_million_chars":0}}`), 0o600))
	remote.unchanged = true
	require.NoError(t, service.syncWithRemote())
	require.Equal(t, 0.01, *service.BillingDefaults().WebSearchPricePerCall)
	require.Zero(t, *service.BillingDefaults().AudioTTSPricePerMillionChars)
	require.Zero(t, service.GetModelPricing("claude-test").CacheCreationInputTokenCostAbove1hr)
	require.NoError(t, os.Remove(supplement))
	require.NoError(t, service.syncWithRemote())
	require.Empty(t, service.GetModelPricing("claude-test").ImagePrices)
	require.Equal(t, 0.01, *service.BillingDefaults().WebSearchPricePerCall)
	// 发布后的快照可以独立修改，不能污染服务。
	snap := before
	snap.BillingDefaults.WebSearchPricePerCall = new(float64)
	snap.Data["claude-test"].ImagePrices["1K"] = 99
	require.Empty(t, service.GetModelPricing("claude-test").ImagePrices)
}

// TestIncompleteCatalogFieldsAreNotReplaced 验证缺一侧报价时也保留目录已有字段。
func TestIncompleteCatalogFieldsAreNotReplaced(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "prices.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"partial":{"input_cost_per_token":99,"output_cost_per_token":0}}`), 0o600))
	remote := &catalogRemoteFixture{body: []byte(`{"providers":{"openai":{"models":{"partial":{"cost":{"input":2},"reasoning":false,"modalities":{"output":[]}}}}}}`)}
	service := NewService(Options{DataDir: dir, RemoteURL: "https://models.dev/catalog.json", FallbackFile: file}, remote)
	require.NoError(t, service.ForceUpdate())
	price := service.GetModelPricing("partial")
	require.InDelta(t, 2e-6, price.InputCostPerToken, 1e-12)
	require.Zero(t, price.OutputCostPerToken)
	require.False(t, price.TokenPricingAbsent)
	require.False(t, *service.ModelAttributes("partial").Reasoning)
	require.Empty(t, *service.ModelAttributes("partial").OutputModalities)
}

// TestShippedSupplementsAreMinimalAndDocumented 验证分发数据没有混入旧展示字段。
func TestShippedSupplementsAreMinimalAndDocumented(t *testing.T) {
	body := modelcatalog.PricingSupplements()
	var records map[string]map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &records))
	for model, fields := range records {
		if model == pricing.BillingDefaultsKey {
			require.Contains(t, fields, "sources")
			continue
		}
		require.Contains(t, fields, "source_url", model)
		require.Contains(t, fields, "verified_at", model)
		for _, field := range []string{"max_input_tokens", "max_output_tokens", "max_tokens", "supports_vision", "supported_endpoints", "supported_modalities"} {
			require.NotContains(t, fields, field, model)
		}
	}
}

// TestSupplementRejectsInvalidOperationAndRule 保证补丁无基础价或被目录覆盖时也不能跳过校验。
func TestSupplementRejectsInvalidOperationAndRule(t *testing.T) {
	for _, body := range []string{
		`{"_billing_defaults":{"web_search_price_per_call":-1}}`,
		`{"_billing_defaults":{"audio_tts_price_per_million_chars":"bad"}}`,
		`{"_billing_defaults":null}`,
		`{"claude-test":{"input_cost_per_token":-1}}`,
		`{"claude-test":{"fast_multiplier":0}}`,
		`{"claude-test":{"input_cost_per_token":1e308,"cache_write_multiplier":1e308}}`,
		`{"claude-test":{"video_prices":{"4K":1}}}`,
		`{"claude-test":{"image_prices":{"1K":null}}}`,
		`{"claude-test":{"time_pricing":{"timezone":"Local","periods":[]}}}`,
	} {
		t.Run(body, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "supplement.json")
			require.NoError(t, os.WriteFile(file, []byte(body), 0o600))
			_, err := loadLocalPricingEntries(file)
			require.Error(t, err)
		})
	}
}

// TestModelRulesReturnIndependentValues 防止调用方修改可空倍率或分时数组污染已发布目录。
func TestModelRulesReturnIndependentValues(t *testing.T) {
	entries, diagnostics, err := pricing.ParsePricingEntries(map[string]json.RawMessage{"model": json.RawMessage(`{"input_cost_per_token":0.000001,"output_cost_per_token":0.000002,"cache_write_multiplier":1.25,"cache_write_1h_multiplier":2,"fast_multiplier":3,"flex_multiplier":0.5,"max_reasoning_effort_multiplier":4,"time_pricing":{"timezone":"UTC","periods":[{"start_time":"01:00","end_time":"03:00","multiplier":2}]}}`)})
	require.NoError(t, err)
	require.NoError(t, diagnostics.ValidationError())
	service := NewServiceFromSnapshot(Options{}, nil, Snapshot{Data: entries})
	calc := billing.NewCalculator(service, billing.CalculatorOptions{})
	first, err := calc.GetModelPricing("model")
	require.NoError(t, err)
	*first.FastMultiplier = 99
	first.TimePricing.Periods[0].Multiplier = 99
	second, err := calc.GetModelPricing("model")
	require.NoError(t, err)
	require.Equal(t, 3.0, *second.FastMultiplier)
	require.Equal(t, 2.0, second.TimePricing.Periods[0].Multiplier)
	require.InDelta(t, 1.25e-6, second.CacheCreation5mPrice, 1e-12)
	require.InDelta(t, 2e-6, second.CacheCreation1hPrice, 1e-12)
}
