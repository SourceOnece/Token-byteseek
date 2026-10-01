package pricing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 显式免费图片缓存不能按文本缓存扣费；缺失专用单价才使用原来的通用价。
func TestImageCacheExplicitZero(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		price := &ModelPricing{CacheReadPricePerToken: 1e-6, ImageCacheReadPriceExplicit: explicit}
		cost := ComputeTokenBreakdown(price, UsageTokens{CacheReadTokens: 100, ImageCacheReadTokens: 100}, 1, "", false)
		if explicit {
			require.Zero(t, cost.CacheReadCost)
		} else {
			require.InDelta(t, .0001, cost.CacheReadCost, 1e-12)
		}
	}
}

// 原生图片缓存应与文本缓存分别计价，不能双计图片输入。
func TestNativeImageCacheCost(t *testing.T) {
	pricing := &ModelPricing{InputPricePerToken: 5e-6, ImageInputPricePerToken: 8e-6, CacheReadPricePerToken: 1.25e-6, ImageCacheReadPricePerToken: 2e-6, ImageOutputPricePerToken: 30e-6}
	computed := ComputeTokenBreakdown(pricing, UsageTokens{InputTokens: 50, ImageInputTokens: 40, CacheReadTokens: 50, ImageCacheReadTokens: 40, OutputTokens: 200, ImageOutputTokens: 200}, 1, "", false)
	require.InDelta(t, 10*5e-6, computed.InputCost, 1e-12)
	require.InDelta(t, 40*8e-6, computed.ImageInputCost, 1e-12)
	require.InDelta(t, 10*1.25e-6+40*2e-6, computed.CacheReadCost, 1e-12)
	require.InDelta(t, 200*30e-6, computed.ImageOutputCost, 1e-12)
	require.InDelta(t, 10*5e-6+40*8e-6+10*1.25e-6+40*2e-6+200*30e-6, computed.TotalCost, 1e-12)
}
