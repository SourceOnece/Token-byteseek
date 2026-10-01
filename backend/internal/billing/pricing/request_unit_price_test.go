package pricing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRequestPriceSeparatesLabelsAndContext 不把空标签当作尺寸，也不把尺寸价用作上下文回退。
func TestRequestPriceSeparatesLabelsAndContext(t *testing.T) {
	zero, low, high := 0.0, 0.03, 0.1
	maxTokens := 128000
	resolved := &ResolvedPricing{Mode: BillingModePerRequest, RequestTiers: []PricingInterval{
		{TierLabel: "1K", PerRequestPrice: &zero},
		{MinTokens: 0, MaxTokens: &maxTokens, PerRequestPrice: &low},
		{MinTokens: maxTokens, PerRequestPrice: &high},
	}}
	_, ok := GetRequestTierPriceValue(resolved, "")
	require.False(t, ok)
	count := 200000
	value, ok := ResolveRequestUnitPrice(resolved, "2K", &count)
	require.True(t, ok)
	require.Equal(t, high, value)
	value, ok = ResolveRequestUnitPrice(resolved, "1K", nil)
	require.True(t, ok)
	require.Zero(t, value)
	_, ok = ResolveRequestUnitPrice(resolved, "2K", nil)
	require.False(t, ok)
}

// TestRequestMissingDefaultIsNotFree 区分显式免费、倍率折为零及没有默认按次价。
func TestRequestMissingDefaultIsNotFree(t *testing.T) {
	count := 0
	_, ok := ResolveRequestUnitPrice(&ResolvedPricing{Mode: BillingModePerRequest}, "", &count)
	require.False(t, ok)
	price, ok := ResolveRequestUnitPrice(&ResolvedPricing{Mode: BillingModePerRequest, DefaultPerRequestPricePresent: true}, "", &count)
	require.True(t, ok)
	require.Zero(t, price)
	unit, multiplier := 0.2, 0.0
	resolved := ResolvePriceCards(&ModelPricingEntry{BillingMode: BillingModePerRequest, PerRequestPrice: &unit, PriceMultiplier: &multiplier}, nil, PricingSourceUnpriced, true)
	price, ok = ResolveRequestUnitPrice(resolved, "", &count)
	require.True(t, ok)
	require.Zero(t, price)
}

// TestImageModeKeepsExplicitDefaultPrice 图片模式的固定默认价不受无标签区间字段影响。
func TestImageModeKeepsExplicitDefaultPrice(t *testing.T) {
	base, tier := 0.2, 0.1
	resolved := ResolvePriceCards(&ModelPricingEntry{BillingMode: BillingModeImage, PerRequestPrice: &base, Intervals: []PricingInterval{{MinTokens: 100, PerRequestPrice: &tier}}}, nil, PricingSourceUnpriced, true)
	price, found := ConfiguredImageUnitPrice(resolved, "2K")
	require.True(t, found)
	require.Equal(t, base, price)
}

// TestConstantRequestIntervalsKeepExplicitFreeQuote 相同区间价可以安全投影，不能把整张免费价卡误报缺价。
func TestConstantRequestIntervalsKeepExplicitFreeQuote(t *testing.T) {
	base, tier, multiplier := 0.2, 0.4, 0.0
	resolved := ResolvePriceCards(&ModelPricingEntry{BillingMode: BillingModePerRequest, PerRequestPrice: &base, PriceMultiplier: &multiplier, Intervals: []PricingInterval{{MinTokens: 0, PerRequestPrice: &tier}}}, nil, PricingSourceUnpriced, true)
	price, ok := ResolveRequestUnitPrice(resolved, "2K", nil)
	require.True(t, ok)
	require.Zero(t, price)
}
