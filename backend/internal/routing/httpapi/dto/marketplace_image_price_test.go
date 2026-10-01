package dto

import (
	"encoding/json"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/stretchr/testify/require"
)

// TestImagePriceJSONPreservesFreeAndMissingSizes 防止部分尺寸免费时把其他尺寸也显示为免费。
func TestImagePriceJSONPreservesFreeAndMissingSizes(t *testing.T) {
	row := modelMarketplacePricingFromRouting(pricing.ModelDisplayPricing{PricingMode: "image", PriceStatus: "priced", ImagePriceSizes: []string{"1K", "2K"}, ImagePrice1K: 0, ImagePrice2K: 0.1})
	body, err := json.Marshal(row)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(body, &fields))
	require.Contains(t, fields, "image_price_1k")
	require.Equal(t, 0.0, fields["image_price_1k"])
	require.Equal(t, 0.1, fields["image_price_2k"])
	require.NotContains(t, fields, "image_price_4k")
}
