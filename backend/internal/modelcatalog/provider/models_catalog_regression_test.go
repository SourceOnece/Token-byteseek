package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/stretchr/testify/require"
)

// catalogPriceForTest 通过生产价格转换验证目录查价结果，避免只断言原始 JSON。
func catalogPriceForTest(t *testing.T, service *Service, model string) *pricing.ModelPricing {
	t.Helper()
	value, _, err := pricing.ResolveModelPricing(model, service.GetModelPricing(model), nil, pricing.ModelPolicy{})
	require.NoError(t, err)
	return value
}

func TestModelsCatalogEmbeddingDefaultPricing(t *testing.T) {
	service := NewService(Options{DataDir: t.TempDir()}, nil)
	require.NoError(t, service.Initialize())
	for _, tc := range []struct {
		model string
		price float64
	}{
		{"text-embedding-3-small", 0.02},
		{"text-embedding-3-large", 0.13},
		{"text-embedding-ada-002", 0.1},
	} {
		t.Run(tc.model, func(t *testing.T) {
			value := catalogPriceForTest(t, service, tc.model)
			cost := pricing.ComputeTokenBreakdown(value, pricing.UsageTokens{InputTokens: 1000000}, 1, "", true)
			require.InDelta(t, tc.price, cost.TotalCost, 1e-12)
		})
	}
}

func TestModelsCatalogExplicitLongContextOverrides(t *testing.T) {
	for _, tc := range []struct {
		name       string
		patch      string
		inputPrice float64
		intervals  int
	}{
		{"inherit", `{}`, 9, 3},
		{"disable", `{"long_context_input_token_threshold":0,"long_context_input_cost_multiplier":1,"long_context_output_cost_multiplier":1}`, 3, 0},
		{"replace", `{"long_context_input_token_threshold":150,"long_context_input_cost_multiplier":2,"long_context_output_cost_multiplier":1.5}`, 6, 2},
		{"delete override", `{"long_context_input_token_threshold":null}`, 9, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			patch := filepath.Join(dir, "override.json")
			require.NoError(t, os.WriteFile(patch, []byte(`{"claude-test":`+tc.patch+`}`), 0o600))
			service := NewService(Options{DataDir: dir, RemoteURL: "https://models.dev/catalog.json", OverrideFile: patch}, &catalogRemoteFixture{body: []byte(modelsCatalogFixture)})
			require.NoError(t, service.ForceUpdate())
			value := catalogPriceForTest(t, service, "claude-test")
			cost := pricing.ComputeTokenBreakdown(value, pricing.UsageTokens{InputTokens: 201}, 1, "", true)
			require.InDelta(t, 201*tc.inputPrice/1e6, cost.TotalCost, 1e-12)
			intervals := pricing.LongContextDisplayPricingIntervals(value, 1)
			require.Len(t, intervals, tc.intervals)
			if tc.intervals > 0 {
				require.InDelta(t, tc.inputPrice/1e6, intervals[len(intervals)-1].InputPricePerToken, 1e-12)
			}
		})
	}
}

const mediaAliasFixture = `{
	"models":{"openai/gpt-image-2":{"name":"Image"}},
	"providers":{
		"openai":{"models":{
			"gpt-image-2":{"cost":{"input":5,"output":30}},
			"gpt-image-2-snapshot":{"canonical_model_id":"openai/gpt-image-2","cost":{"input":4,"output":20}}
		}},
		"openrouter":{"models":{"gpt-image-2":{"canonical_model_id":"openai/gpt-image-2","cost":{"input":1,"output":2}}}}
	}
}`

func TestModelsCatalogMediaSupplementAliases(t *testing.T) {
	dir := t.TempDir()
	supplement := filepath.Join(dir, "supplement.json")
	patch := filepath.Join(dir, "override.json")
	require.NoError(t, os.WriteFile(supplement, []byte(`{"gpt-image-2":{"input_cost_per_image_token":0.000008,"output_cost_per_image_token":0.00003}}`), 0o600))
	service := NewService(Options{DataDir: dir, RemoteURL: "https://models.dev/catalog.json", FallbackFile: supplement, OverrideFile: patch}, &catalogRemoteFixture{body: []byte(mediaAliasFixture)})
	require.NoError(t, service.ForceUpdate())
	for _, model := range []string{"gpt-image-2", "openai/gpt-image-2"} {
		value := catalogPriceForTest(t, service, model)
		cost := pricing.ComputeTokenBreakdown(value, pricing.UsageTokens{InputTokens: 1000, ImageInputTokens: 1000}, 1, "", true)
		require.InDelta(t, 0.008, cost.TotalCost, 1e-12, model)
		require.Equal(t, "local_supplement", service.GetModelPricing(model).PriceSources["image_input"])
	}
	for _, model := range []string{"openrouter/gpt-image-2", "gpt-image-2-snapshot", "openai/gpt-image-2-snapshot"} {
		require.Zero(t, service.GetModelPricing(model).InputCostPerImageToken, model)
	}
	// 精确名称上的本地价格覆盖仍高于共享的媒体补充。
	require.NoError(t, os.WriteFile(patch, []byte(`{"openai/gpt-image-2":{"input_cost_per_image_token":0.000012}}`), 0o600))
	require.NoError(t, service.ForceUpdate())
	require.InDelta(t, 12e-6, service.GetModelPricing("openai/gpt-image-2").InputCostPerImageToken, 1e-12)
	require.InDelta(t, 8e-6, service.GetModelPricing("gpt-image-2").InputCostPerImageToken, 1e-12)
}

func TestModelsCatalogExactMediaSupplementKeepsExplicitZero(t *testing.T) {
	dir := t.TempDir()
	supplement := filepath.Join(dir, "supplement.json")
	require.NoError(t, os.WriteFile(supplement, []byte(`{
		"gpt-image-2":{"input_cost_per_image_token":0.000008},
		"openai/gpt-image-2":{"input_cost_per_image_token":0}
	}`), 0o600))
	service := NewService(Options{DataDir: dir, RemoteURL: "https://models.dev/catalog.json", FallbackFile: supplement}, &catalogRemoteFixture{body: []byte(mediaAliasFixture)})
	require.NoError(t, service.ForceUpdate())
	require.InDelta(t, 8e-6, service.GetModelPricing("gpt-image-2").InputCostPerImageToken, 1e-12)
	qualified := service.GetModelPricing("openai/gpt-image-2")
	require.True(t, qualified.ImageInputPricePresent)
	require.Zero(t, qualified.InputCostPerImageToken)
}

func TestModelsCatalogInvalidPricePatchKeepsPublishedSnapshot(t *testing.T) {
	dir := t.TempDir()
	patch := filepath.Join(dir, "override.json")
	remote := &catalogRemoteFixture{body: []byte(modelsCatalogFixture)}
	service := NewService(Options{DataDir: dir, RemoteURL: "https://models.dev/catalog.json", OverrideFile: patch}, remote)
	require.NoError(t, service.ForceUpdate())
	before := service.AttributesSnapshot()
	beforePrices := service.Snapshot().Data
	beforeFile, err := os.ReadFile(service.catalogFilePath())
	require.NoError(t, err)
	remote.body = []byte(strings.ReplaceAll(modelsCatalogFixture, `"name":"Claude"`, `"name":"Changed"`))
	for _, tc := range []struct {
		name  string
		patch string
		field string
	}{
		{"base price", `{"input_cost_per_token":"invalid"}`, "input_cost_per_token"},
		{"tier price", `{"context_prices":[{"threshold":100,"pricing":{"input_cost_per_token":"invalid","output_cost_per_token":0.000015}}]}`, "input_cost_per_token"},
		{"threshold", `{"long_context_input_token_threshold":"invalid"}`, "long_context_input_token_threshold"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(patch, []byte(`{"claude-test":`+tc.patch+`}`), 0o600))
			err := service.ForceUpdate()
			require.Error(t, err)
			require.ErrorContains(t, err, "claude-test")
			require.ErrorContains(t, err, tc.field)
			after := service.AttributesSnapshot()
			require.Equal(t, before.Version, after.Version)
			require.Equal(t, before.LastUpdated, after.LastUpdated)
			require.Equal(t, before.Items, after.Items)
			require.Equal(t, beforePrices, service.Snapshot().Data)
			require.Contains(t, after.LastError, tc.field)
			afterFile, err := os.ReadFile(service.catalogFilePath())
			require.NoError(t, err)
			require.Equal(t, beforeFile, afterFile)
		})
	}
	// 合法的零价可以正常发布，并清除之前的错误。
	require.NoError(t, os.WriteFile(patch, []byte(`{"claude-test":{"input_cost_per_token":0}}`), 0o600))
	require.NoError(t, service.ForceUpdate())
	require.Zero(t, service.GetModelPricing("claude-test").InputCostPerToken)
	require.Empty(t, service.AttributesSnapshot().LastError)
	require.NotNil(t, service.ModelAttributes("attributes-only").DisplayName)
}

func TestModelsCatalogInvalidLocalReloadReportsError(t *testing.T) {
	dir := t.TempDir()
	patch := filepath.Join(dir, "override.json")
	remote := &catalogRemoteFixture{body: []byte(modelsCatalogFixture), etag: "v1"}
	service := NewService(Options{DataDir: dir, RemoteURL: "https://models.dev/catalog.json", OverrideFile: patch}, remote)
	require.NoError(t, service.ForceUpdate())
	before := service.Snapshot().Data
	require.NoError(t, os.WriteFile(patch, []byte(`{"claude-test":{"input_cost_per_token":"invalid"}}`), 0o600))
	// 304 只确认远程未变，同步入口也必须报告本地覆盖的校验错误。
	remote.unchanged = true
	require.ErrorContains(t, service.syncWithRemote(), "input_cost_per_token")
	service.reloadIfCustomFilesChanged()
	require.Contains(t, service.AttributesSnapshot().LastError, "input_cost_per_token")
	require.Equal(t, before, service.Snapshot().Data)
}

// TestModelsCatalogGeminiImageTextPricing 同时验证内嵌目录、媒体补充和实际用量拆分。
func TestModelsCatalogGeminiImageTextPricing(t *testing.T) {
	service := NewService(Options{
		DataDir:      t.TempDir(),
		FallbackFile: "../../../resources/model-pricing/model_pricing_supplements.json",
	}, nil)
	require.NoError(t, service.Initialize())
	for _, tc := range []struct {
		model                 string
		textPrice, imagePrice float64
	}{
		{"gemini-3-pro-image", 12e-6, 120e-6},
		{"gemini-3-pro-image-preview", 12e-6, 120e-6},
		{"gemini-2.5-flash-image", 2.5e-6, 30e-6},
		{"gemini-3.1-flash-image", 3e-6, 60e-6},
		{"gemini-3.1-flash-image-preview", 3e-6, 60e-6},
		{"gemini-3.1-flash-lite-image", 2.5e-6, 30e-6},
	} {
		for _, model := range []string{tc.model, "google/" + tc.model} {
			t.Run(model, func(t *testing.T) {
				value := catalogPriceForTest(t, service, model)
				cost := pricing.ComputeTokenBreakdown(value, pricing.UsageTokens{OutputTokens: 2000, ImageOutputTokens: 1000}, 1, "", true)
				require.InDelta(t, 1000*tc.textPrice, cost.OutputCost, 1e-12)
				require.InDelta(t, 1000*tc.imagePrice, cost.ImageOutputCost, 1e-12)
				display := pricing.BuildTokenDisplayPricing(value, 1)
				require.InDelta(t, tc.textPrice, display.OutputPricePerToken, 1e-12)
				require.InDelta(t, tc.imagePrice, display.ImageOutputPricePerToken, 1e-12)
				require.Equal(t, "local_supplement", service.GetModelPricing(model).PriceSources["output"])
			})
		}
	}
}

func TestModelsCatalogGeminiImageSupplementPrecedence(t *testing.T) {
	dir := t.TempDir()
	supplement := filepath.Join(dir, "supplement.json")
	patch := filepath.Join(dir, "override.json")
	const fixture = `{"providers":{
		"google":{"models":{"gemini-image-test":{"cost":{"input":2,"output":150},"modalities":{"output":["text","image"]}}}},
		"openrouter":{"models":{"gemini-image-test":{"cost":{"input":1,"output":9},"modalities":{"output":["text","image"]}}}}
	}}`
	service := NewService(Options{
		DataDir:      dir,
		RemoteURL:    "https://models.dev/catalog.json",
		FallbackFile: supplement,
		OverrideFile: patch,
	}, &catalogRemoteFixture{body: []byte(fixture)})
	require.NoError(t, service.ForceUpdate())
	// 缺少文本费率时保持未定价，不能把图片费率或缺失值当作文本价。
	_, _, err := pricing.ResolveModelPricing("gemini-image-test", service.GetModelPricing("gemini-image-test"), nil, pricing.ModelPolicy{})
	require.ErrorIs(t, err, pricing.ErrModelPricingUnavailable)
	require.InDelta(t, 150e-6, service.GetModelPricing("gemini-image-test").OutputCostPerImageToken, 1e-12)
	require.NoError(t, os.WriteFile(supplement, []byte(`{"gemini-image-test":{"output_cost_per_token":0,"output_cost_per_image_token":0.00012}}`), 0o600))
	require.NoError(t, service.ForceUpdate())
	for _, model := range []string{"gemini-image-test", "google/gemini-image-test"} {
		value := catalogPriceForTest(t, service, model)
		require.Zero(t, value.OutputPricePerToken)
		require.InDelta(t, 150e-6, value.ImageOutputPricePerToken, 1e-12)
	}
	// 原厂文本补充不跨到中继报价；管理员精确覆盖仍具有最高优先级。
	require.InDelta(t, 9e-6, catalogPriceForTest(t, service, "openrouter/gemini-image-test").OutputPricePerToken, 1e-12)
	require.NoError(t, os.WriteFile(patch, []byte(`{"google/gemini-image-test":{"output_cost_per_token":0.000007,"output_cost_per_image_token":0.00008,"output_cost_per_image":0}}`), 0o600))
	require.NoError(t, service.ForceUpdate())
	value := catalogPriceForTest(t, service, "google/gemini-image-test")
	require.InDelta(t, 7e-6, value.OutputPricePerToken, 1e-12)
	require.InDelta(t, 80e-6, value.ImageOutputPricePerToken, 1e-12)
	require.Equal(t, "local_override", service.GetModelPricing("google/gemini-image-test").PriceSources["output"])
	raw := service.GetModelPricing("google/gemini-image-test")
	require.True(t, raw.ImagePricePresent)
	unit, found := pricing.DefaultImagePrice(raw, pricing.ImageBillingSize1K)
	require.True(t, found)
	require.Zero(t, unit)
	require.Equal(t, "local_override", raw.PriceSources["image"])
}

func TestModelsCatalogImagePriceOverrideAcrossContextTiers(t *testing.T) {
	service := NewService(Options{DataDir: t.TempDir()}, nil)
	require.NoError(t, service.Initialize())
	base := catalogPriceForTest(t, service, "gpt-5.4")
	require.NotEmpty(t, base.ContextPrices)
	imagePrice := 1e-6
	resolved := pricing.ResolvePriceCards(&pricing.ModelPricingEntry{ImageInputPrice: &imagePrice}, base, pricing.PricingSourceCatalog, true)
	for _, input := range []int{10000, 272000, 272001, 300000} {
		cost, err := pricing.CalculateTokenCost(resolved, pricing.CostInput{
			Model:          "gpt-5.4",
			Tokens:         pricing.UsageTokens{InputTokens: input, ImageInputTokens: 1000},
			RateMultiplier: 1,
		})
		require.NoError(t, err)
		require.InDelta(t, 0.001, cost.ImageInputCost, 1e-12, "input=%d", input)
	}
}

// TestModelsCatalogGrokInclusiveContextBoundary 覆盖实际目录的阈值前、阈值处及缓存混合输入。
func TestModelsCatalogGrokInclusiveContextBoundary(t *testing.T) {
	service := NewService(Options{DataDir: t.TempDir()}, nil)
	require.NoError(t, service.Initialize())
	for _, model := range []string{"grok-4.6", "xai/grok-4.6"} {
		t.Run(model, func(t *testing.T) {
			base := catalogPriceForTest(t, service, model)
			for _, tc := range []struct {
				tokens pricing.UsageTokens
				want   float64
				long   bool
			}{
				{pricing.UsageTokens{InputTokens: 199999, OutputTokens: 1000}, 0.405998, false},
				{pricing.UsageTokens{InputTokens: 200000, OutputTokens: 1000}, 0.812, true},
				{pricing.UsageTokens{InputTokens: 200001, OutputTokens: 1000}, 0.812004, true},
				{pricing.UsageTokens{InputTokens: 100000, CacheReadTokens: 100000, OutputTokens: 1000}, 0.512, true},
			} {
				cost := pricing.ComputeTokenBreakdown(base, tc.tokens, 1, "", true)
				require.InDelta(t, tc.want, cost.TotalCost, 1e-12)
				require.Equal(t, tc.long, cost.LongContextBillingApplied)
			}
			intervals := pricing.LongContextDisplayPricingIntervals(base, 1)
			require.Len(t, intervals, 2)
			require.Equal(t, 199999, *intervals[0].MaxTokens)
			require.Equal(t, 199999, intervals[1].MinTokens)
			cost := pricing.ComputeTokenBreakdown(base, pricing.UsageTokens{InputTokens: 200000, OutputTokens: 1000}, 1, "", false)
			require.InDelta(t, 0.406, cost.TotalCost, 1e-12)
		})
	}
}

func TestModelsCatalogExplicitZeroImageOutput(t *testing.T) {
	dir := t.TempDir()
	supplement := filepath.Join(dir, "supplement.json")
	patch := filepath.Join(dir, "override.json")
	require.NoError(t, os.WriteFile(supplement, []byte(`{"gemini-image-test":{"output_cost_per_token":0.000012}}`), 0o600))
	remote := &catalogRemoteFixture{body: []byte(`{"providers":{"google":{"models":{"gemini-image-test":{"cost":{"input":2,"output":0},"modalities":{"output":["text","image"]}}}}}}`)}
	service := NewService(Options{
		DataDir:      dir,
		RemoteURL:    "https://models.dev/catalog.json",
		FallbackFile: supplement,
		OverrideFile: patch,
	}, remote)
	for _, fromOverride := range []bool{false, true} {
		if fromOverride {
			remote.body = []byte(strings.ReplaceAll(string(remote.body), `"output":0`, `"output":120`))
			require.NoError(t, os.WriteFile(patch, []byte(`{"gemini-image-test":{"output_cost_per_image_token":0}}`), 0o600))
		}
		require.NoError(t, service.ForceUpdate())
		base := catalogPriceForTest(t, service, "gemini-image-test")
		cost := pricing.ComputeTokenBreakdown(base, pricing.UsageTokens{OutputTokens: 2000, ImageOutputTokens: 1000}, 1, "", true)
		require.InDelta(t, 0.012, cost.OutputCost, 1e-12)
		require.Zero(t, cost.ImageOutputCost)
		require.True(t, base.ImageOutputPriceExplicit)
	}
	// 删除价格桶后恢复缺价回退，不能把缺失误当成显式免费。
	require.NoError(t, os.WriteFile(patch, []byte(`{"gemini-image-test":{"output_cost_per_image_token":null}}`), 0o600))
	require.NoError(t, service.ForceUpdate())
	base := catalogPriceForTest(t, service, "gemini-image-test")
	require.False(t, base.ImageOutputPriceExplicit)
	cost := pricing.ComputeTokenBreakdown(base, pricing.UsageTokens{OutputTokens: 1000, ImageOutputTokens: 1000}, 1, "", true)
	require.InDelta(t, 0.012, cost.ImageOutputCost, 1e-12)
}

func TestModelsCatalogMistralNativeAlias(t *testing.T) {
	service := NewService(Options{DataDir: t.TempDir()}, nil)
	require.NoError(t, service.Initialize())
	plain := catalogPriceForTest(t, service, "devstral-latest")
	qualified := catalogPriceForTest(t, service, "mistral/devstral-latest")
	require.InDelta(t, 0.4e-6, plain.InputPricePerToken, 1e-12)
	require.Equal(t, qualified, plain)
	require.Equal(t, service.ModelAttributes("mistral/devstral-latest"), service.ModelAttributes("devstral-latest"))
	require.NotNil(t, service.ModelAttributes("devstral-latest").Context)
	// 中继记录保持自己的价格与属性。
	require.InDelta(t, 0.44e-6, catalogPriceForTest(t, service, "requesty/devstral-latest").InputPricePerToken, 1e-12)
}

// TestModelsCatalogOfflineManualUpdate 验证磁盘缓存不存在时，手动更新仍能使用已加载的内存目录。
func TestModelsCatalogOfflineManualUpdate(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(dir, "cache")
	patch := filepath.Join(dir, "override.json")
	// 用不可作为目录的路径稳定模拟缓存不可写，测试不依赖进程的文件权限。
	require.NoError(t, os.WriteFile(cache, []byte("unwritable cache"), 0o600))
	require.NoError(t, os.WriteFile(patch, []byte(`{"gpt-5.4":{"input_cost_per_token":0.000007}}`), 0o600))
	service := NewService(Options{DataDir: cache, OverrideFile: patch}, nil)
	require.NoError(t, service.Initialize())
	before := service.AttributesSnapshot()
	require.NoError(t, os.WriteFile(patch, []byte(`{"gpt-5.4":{"input_cost_per_token":0.000009}}`), 0o600))
	require.NoError(t, service.ForceUpdate())
	require.InDelta(t, 9e-6, service.GetModelPricing("gpt-5.4").InputCostPerToken, 1e-12)
	require.Equal(t, before.Version, service.AttributesSnapshot().Version)
	require.Empty(t, service.AttributesSnapshot().LastError)
	// 本地更新同样完整校验，失败时保留上一次成功价格和属性。
	require.NoError(t, os.WriteFile(patch, []byte(`{"gpt-5.4":{"input_cost_per_token":"invalid"}}`), 0o600))
	require.Error(t, service.ForceUpdate())
	require.InDelta(t, 9e-6, service.GetModelPricing("gpt-5.4").InputCostPerToken, 1e-12)
	require.Equal(t, before.Items, service.AttributesSnapshot().Items)
	require.NotEmpty(t, service.AttributesSnapshot().LastError)
	require.NoError(t, os.WriteFile(patch, []byte(`{"gpt-5.4":{"input_cost_per_token":0}}`), 0o600))
	require.NoError(t, service.ForceUpdate())
	require.Zero(t, service.GetModelPricing("gpt-5.4").InputCostPerToken)
	require.Empty(t, service.AttributesSnapshot().LastError)
}
