package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestModelsCatalogFallbackIsStable 使用实际目录验证相同用量始终按原厂完整价卡结算。
func TestModelsCatalogFallbackIsStable(t *testing.T) {
	service := NewService(Options{DataDir: t.TempDir()}, nil)
	require.NoError(t, service.Initialize())
	for _, model := range []string{"claude-opus-4-5", "claude-opus-4-6"} {
		require.NotNil(t, service.GetModelPricing(model))
		require.Nil(t, service.GetModelPricing(model+"-20990101"))
		require.Nil(t, service.ModelAttributes(model+"-20990101").Context)
	}
}

const fallbackOriginFixture = `{"providers":{
	"anthropic":{"models":{
		"claude-opus-4-6":{"cost":{"input":5,"output":25}},
		"claude-opus-4-6-20260101":{"cost":{"input":6,"output":30}}
	}},
	"openai":{"models":{"gpt-5.4":{"cost":{"input":2.5,"output":15}}}},
	"relay":{"models":{
		"claude-opus-4-6":{"cost":{"input":1,"output":2}},
		"claude-opus-4-6-discount":{"cost":{"input":0,"output":0}},
		"gpt-5.4":{"cost":{"input":1,"output":3}}
	}}
}}`

func TestModelsCatalogFallbackOriginAndOverrides(t *testing.T) {
	dir := t.TempDir()
	patch := filepath.Join(dir, "override.json")
	service := NewService(Options{
		DataDir:      dir,
		RemoteURL:    "https://models.dev/catalog.json",
		OverrideFile: patch,
	}, &catalogRemoteFixture{body: []byte(fallbackOriginFixture)})
	require.NoError(t, service.ForceUpdate())
	check := func(reader *Service, input float64) {
		require.InDelta(t, input, reader.GetModelPricing("claude-opus-4-6").InputCostPerToken, 1e-12)
		for range 20 {
			for _, model := range []string{"claude-opus-4-6-thinking", "claude-opus-4-6-20990101"} {
				require.Nil(t, reader.GetModelPricing(model))
			}
		}
		require.InDelta(t, 1e-6, reader.GetModelPricing("relay/claude-opus-4-6").InputCostPerToken, 1e-12)
		require.Equal(t, "unpriced", reader.GetModelPricing("relay/claude-opus-4-6-thinking").Source)
		for _, model := range []string{"gpt-5.4-20990101", "gpt-5.4-openai-compact"} {
			require.Nil(t, reader.GetModelPricing(model))
		}
		require.InDelta(t, 1e-6, reader.GetModelPricing("relay/gpt-5.4").InputCostPerToken, 1e-12)
	}
	check(service, 5e-6)
	check(service.ReadOnlySnapshot(), 5e-6)
	require.NoError(t, os.WriteFile(patch, []byte(`{
		"claude-opus-4-6":{"input_cost_per_token":0},
		"claude-opus-4-7":{"input_cost_per_token":0.000008,"output_cost_per_token":0.00004}
	}`), 0o600))
	require.NoError(t, service.ForceUpdate())
	check(service, 0)
	check(service.ReadOnlySnapshot(), 0)
	// 本地新增的明确价格仍可供既有系列规则使用。
	require.InDelta(t, 8e-6, service.GetModelPricing("claude-opus-4-7").InputCostPerToken, 1e-12)
}

func TestModelsCatalogFallbackDoesNotBorrowRelayOnlyModel(t *testing.T) {
	remote := &catalogRemoteFixture{body: []byte(`{"providers":{"relay":{"models":{"claude-opus-4-6":{"cost":{"input":1,"output":2}}}}}}`)}
	service := NewService(Options{DataDir: t.TempDir(), RemoteURL: "https://models.dev/catalog.json"}, remote)
	require.NoError(t, service.ForceUpdate())
	// 唯一来源的显式名称仍可查询；未知变体不因此获得该中继的报价。
	require.NotNil(t, service.GetModelPricing("claude-opus-4-6"))
	require.Nil(t, service.GetModelPricing("claude-opus-4-6-thinking"))
	require.Nil(t, service.ReadOnlySnapshot().GetModelPricing("claude-opus-4-6-thinking"))
}
