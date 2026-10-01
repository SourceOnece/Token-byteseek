package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// newCatalogWithOverride 使用真实离线目录测试本地覆盖，不复制旧加载流程。
func newCatalogWithOverride(t *testing.T, patch string) *Service {
	t.Helper()
	service := newOfflinePricingFixture(t)
	service.options.OverrideFile = filepath.Join(t.TempDir(), "overrides.json")
	require.NoError(t, os.WriteFile(service.options.OverrideFile, []byte(patch), 0o600))
	return service
}

// TestPricingOverride_FieldMerge 验证字段覆盖、删除和提供方旧字段的归一化优先级。
func TestPricingOverride_FieldMerge(t *testing.T) {
	for _, tc := range []struct{ name, patch, provider string }{
		{"legacy", `"litellm_provider":"custom"`, "custom"},
		{"new wins", `"litellm_provider":"legacy","provider":"custom"`, "custom"},
		{"empty wins", `"litellm_provider":"legacy","provider":""`, ""},
		{"null wins", `"litellm_provider":"legacy","provider":null`, ""},
		{"legacy null", `"litellm_provider":null`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := newCatalogWithOverride(t, `{"gpt-5.5":{"input_cost_per_token":0,"cache_read_input_token_cost":null,`+tc.patch+`}}`)
			before := service.GetModelPricing("gpt-5.5")
			require.NoError(t, service.ForceUpdate())
			after := service.GetModelPricing("gpt-5.5")
			require.Zero(t, after.InputCostPerToken)
			require.False(t, after.CacheReadPricePresent)
			require.Equal(t, before.OutputCostPerToken, after.OutputCostPerToken)
			require.Equal(t, tc.provider, after.Provider)
		})
	}
}

// TestPricingOverride_LoadPipeline 验证覆盖层既能修改补充条目，也能添加独立模型。
func TestPricingOverride_LoadPipeline(t *testing.T) {
	service := newHotReloadCatalog(t,
		`{"local-model":{"litellm_provider":"custom","input_cost_per_token":0.000004,"output_cost_per_token":0.000008}}`,
		`{"local-model":{"input_cost_per_token":0.000009},"new-model":{"provider":"custom","input_cost_per_token":0.000005,"output_cost_per_token":0.00001}}`)
	require.InDelta(t, 9e-6, service.GetModelPricing("local-model").InputCostPerToken, 1e-12)
	require.InDelta(t, 8e-6, service.GetModelPricing("local-model").OutputCostPerToken, 1e-12)
	require.InDelta(t, 5e-6, service.GetModelPricing("new-model").InputCostPerToken, 1e-12)
	require.Equal(t, "custom", service.GetModelPricing("local-model").Provider)
}

// TestPricingOverride_IneffectiveEntryWarns 验证拼错的模型补丁不会静默成功。
func TestPricingOverride_IneffectiveEntryWarns(t *testing.T) {
	sink, restore := captureStructuredLog(t)
	defer restore()
	service := newHotReloadCatalog(t, "", `{"typo-model":{"long_context_input_token_threshold":0}}`)
	require.NotContains(t, service.Snapshot().Data, "typo-model")
	require.True(t, sink.ContainsMessageAtLevel("override had no effect for 1 model(s): typo-model", "warn"))
}

// TestPricingOverride_InvalidLayerKeepsSnapshot 验证顶层 null、非对象条目和非法字段均保留整个发布版本。
func TestPricingOverride_InvalidLayerKeepsSnapshot(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `{invalid`, `{"remote-model":null}`, `{"remote-model":"oops"}`, `{"remote-model":{"input_cost_per_token":"bad"}}`} {
		for _, layer := range []string{"override", "supplement"} {
			t.Run(layer+body, func(t *testing.T) {
				service := newHotReloadCatalog(t, `{}`, `{}`)
				before := service.Snapshot()
				attrs := service.AttributesSnapshot()
				disk := readCatalogTestFile(t, service.catalogFilePath())
				path := service.options.OverrideFile
				if layer == "supplement" {
					path = service.options.FallbackFile
				}
				require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
				require.Error(t, service.ForceUpdate())
				require.Equal(t, before.Data, service.Snapshot().Data)
				require.Equal(t, attrs.Items, service.AttributesSnapshot().Items)
				require.Equal(t, before.LastUpdated, service.Snapshot().LastUpdated)
				require.Equal(t, before.LocalHash, service.Snapshot().LocalHash)
				require.Equal(t, disk, readCatalogTestFile(t, service.catalogFilePath()))
				require.NotEmpty(t, service.AttributesSnapshot().LastError)
				require.NoError(t, os.Remove(path))
				require.NoError(t, service.ForceUpdate())
				require.Empty(t, service.AttributesSnapshot().LastError)
			})
		}
	}
}

// TestPricingOverride_DisablesGPT55LadderOnDefaultCatalog 验证显式零阈值关闭绝对阶梯，删除覆盖后恢复目录规则。
func TestPricingOverride_DisablesGPT55LadderOnDefaultCatalog(t *testing.T) {
	service := newCatalogWithOverride(t, `{"gpt-5.5":{"long_context_input_token_threshold":0}}`)
	require.NotEmpty(t, service.GetModelPricing("gpt-5.5").ContextPrices)
	require.NoError(t, service.ForceUpdate())
	price, err := newBillingFixture(service).GetModelPricing("gpt-5.5")
	require.NoError(t, err)
	require.Empty(t, price.ContextPrices)
	require.Zero(t, price.LongContextInputThreshold)
	require.InDelta(t, 5e-6, price.InputPricePerToken, 1e-12)
	require.NoError(t, os.WriteFile(service.options.OverrideFile, []byte(`{"gpt-5.5":{"long_context_input_token_threshold":null}}`), 0o600))
	require.NoError(t, service.ForceUpdate())
	require.NotEmpty(t, service.GetModelPricing("gpt-5.5").ContextPrices)
}
