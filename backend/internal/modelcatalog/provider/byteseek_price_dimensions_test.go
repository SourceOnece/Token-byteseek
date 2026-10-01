package provider

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/modelcatalog"
	"github.com/stretchr/testify/require"
)

// 使用发布资源验证真实目录合并，图片缓存桶不能退回文本缓存单价。
func TestByteSeekImageCacheSupplement(t *testing.T) {
	s := NewService(Options{FallbackFile: "../../../resources/model-pricing/model_pricing_supplements.json"}, nil)
	body, err := modelcatalog.Offline()
	require.NoError(t, err)
	_, prices, err := s.buildModelsCatalog(body)
	require.NoError(t, err)
	for _, model := range []string{"gpt-image-1.5", "gpt-image-2", "gpt-image-2.5-flare"} {
		require.NotNil(t, prices[model], model)
		require.InDelta(t, 2e-6, prices[model].CacheReadInputImageTokenCost, 1e-12, model)
	}
}

// 已有管理员显式零价不能被补充层覆盖。
func TestByteSeekImageCacheOverridePrecedence(t *testing.T) {
	s := newHotReloadCatalog(t, `{"remote-model":{"cache_read_input_image_token_cost":0.000002}}`, `{"remote-model":{"cache_read_input_image_token_cost":0}}`)
	require.Zero(t, s.GetModelPricing("remote-model").CacheReadInputImageTokenCost)
}
