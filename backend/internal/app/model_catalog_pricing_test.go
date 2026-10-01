package app

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/TokenFlux/TokenRouter/internal/modelcatalog/provider"
	"github.com/stretchr/testify/require"
)

// TestModelsCatalogMediaQuotes 使用实际离线目录和管理页装配，区分输出能力与计费单位。
func TestModelsCatalogMediaQuotes(t *testing.T) {
	service := provider.NewService(provider.Options{
		DataDir:      t.TempDir(),
		FallbackFile: "../../resources/model-pricing/model_pricing_supplements.json",
	}, nil)
	require.NoError(t, service.Initialize())
	calculator := billing.NewCalculator(service, billing.CalculatorOptions{})
	snapshot := providePricingCatalog(calculator, service).Snapshot()
	rows := map[string]pricing.DefaultModelPrice{}
	for _, row := range snapshot.Prices {
		rows[row.Model] = row
	}
	for _, model := range []string{"gemini-omni-flash-preview", "deep-research-preview-04-2026", "gpt-image-2"} {
		t.Run(model, func(t *testing.T) {
			row, found := rows[model]
			require.True(t, found)
			require.Equal(t, "token", row.BillingMode)
			require.Equal(t, "priced", row.PriceStatus)
			for _, price := range row.Prices {
				require.NotEqual(t, "USD/s", price.Unit)
				require.NotEqual(t, "USD/image", price.Unit)
			}
		})
	}
	// 已知专用媒体报价与 Gemini 图文 token 费率继续保留。
	require.Equal(t, "video", rows["grok-imagine-video"].BillingMode)
	require.Equal(t, "image", rows["gemini-3-pro-image"].BillingMode)
	values := map[string]float64{}
	for _, price := range rows["gemini-3-pro-image"].Prices {
		if price.Value != nil {
			values[price.Key] = *price.Value
		}
	}
	require.InDelta(t, 12, values["output"], 1e-12)
	require.InDelta(t, 120, values["image_output"], 1e-12)
	require.InDelta(t, 0.134, values["1K"], 1e-12)
	attrs := service.ModelAttributes("gemini-omni-flash-preview")
	require.NotNil(t, attrs.OutputModalities)
	require.Contains(t, *attrs.OutputModalities, "video")
}
