package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/stretchr/testify/require"
)

// TestEmbeddedSupplementsWithoutResources 验证空工作目录、离线启动及缺失外部文件都保留官方价格。
func TestEmbeddedSupplementsWithoutResources(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, path := range []string{"", "./resources/model-pricing/model_pricing_supplements.json"} {
		service := NewService(Options{DataDir: t.TempDir(), FallbackFile: path}, nil)
		require.NoError(t, service.Initialize())
		calculator := billing.NewCalculator(service, billing.CalculatorOptions{})
		price, err := calculator.DefaultImagePrice("gemini-3-pro-image", "1K")
		require.NoError(t, err)
		require.Equal(t, 0.134, price)
		require.Equal(t, 0.02, calculator.CalculateWebSearchCost(2, nil, 1).ActualCost)
		require.Equal(t, 15.0, *service.BillingDefaults().AudioTTSPricePerMillionChars)
		require.NoError(t, service.ForceUpdate())
		require.Equal(t, 0.01, *service.BillingDefaults().WebSearchPricePerCall)
	}
	_, err := os.Stat("resources")
	require.True(t, os.IsNotExist(err), "加载默认价格不应创建外部资源")
}

// TestEmbeddedSupplementsCustomLayer 验证目录优先、自定义零价、官方缺项及删除后的恢复。
func TestEmbeddedSupplementsCustomLayer(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "custom.json")
	custom := []byte(`{"gemini-3-pro-image":{"input_cost_per_token":99,"output_cost_per_token":0,"image_prices":{"1K":0}},"_billing_defaults":{"web_search_price_per_call":0}}`)
	require.NoError(t, os.WriteFile(file, custom, 0o600))
	service := NewService(Options{DataDir: dir, FallbackFile: file}, nil)
	require.NoError(t, service.Initialize())
	for _, name := range []string{"gemini-3-pro-image", "google/gemini-3-pro-image"} {
		price := service.GetModelPricing(name)
		require.NotNil(t, price)
		require.NotEqual(t, 99.0, price.InputCostPerToken, "自定义补充不能覆盖目录已有报价")
		require.Zero(t, price.OutputCostPerToken)
		require.Zero(t, price.ImagePrices["1K"])
	}
	require.Zero(t, *service.BillingDefaults().WebSearchPricePerCall)
	require.Equal(t, 15.0, *service.BillingDefaults().AudioTTSPricePerMillionChars)
	saved, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, custom, saved, "加载不能改写自定义文件")

	before := service.Snapshot()
	require.NoError(t, os.WriteFile(file, []byte(`{"_billing_defaults":null}`), 0o600))
	require.Error(t, service.ForceUpdate())
	require.Equal(t, before.Data, service.Snapshot().Data)
	require.Equal(t, before.BillingDefaults, service.BillingDefaults())

	require.NoError(t, os.Remove(file))
	service.reloadIfCustomFilesChanged()
	for _, name := range []string{"gemini-3-pro-image", "google/gemini-3-pro-image"} {
		price := service.GetModelPricing(name)
		require.Equal(t, 12e-6, price.OutputCostPerToken)
		require.Equal(t, 0.134, price.ImagePrices["1K"])
	}
	require.Equal(t, 0.01, *service.BillingDefaults().WebSearchPricePerCall)
}

// TestEmbeddedSupplementsSurviveInvalidCustomFile 验证首次启动遇到损坏文件时仍保留官方补充。
func TestEmbeddedSupplementsSurviveInvalidCustomFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "custom.json")
	require.NoError(t, os.WriteFile(file, []byte(`{`), 0o600))
	service := NewService(Options{DataDir: dir, FallbackFile: file}, nil)
	require.NoError(t, service.Initialize())
	require.NotEmpty(t, service.AttributesSnapshot().LastError)
	require.Equal(t, 0.134, service.GetModelPricing("gemini-3-pro-image").ImagePrices["1K"])
	require.Equal(t, 0.01, *service.BillingDefaults().WebSearchPricePerCall)
}
