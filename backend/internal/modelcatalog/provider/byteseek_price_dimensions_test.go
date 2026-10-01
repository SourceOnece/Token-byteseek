package provider

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/modelcatalog"
	"github.com/stretchr/testify/require"
)

// 使用发布资源验证真实目录合并，图片缓存桶不能退回文本缓存单价。
func TestByteSeekImageCacheSupplement(t *testing.T) {
	s := NewService(Options{}, nil)
	body, err := modelcatalog.Offline()
	require.NoError(t, err)
	_, prices, _, err := s.buildModelsCatalog(body)
	require.NoError(t, err)
	for _, model := range []string{"gpt-image-1.5", "gpt-image-2", "gpt-image-2.5-flare"} {
		require.NotNil(t, prices[model], model)
		require.InDelta(t, 2e-6, prices[model].CacheReadInputImageTokenCost, 1e-12, model)
	}
}

// 自定义补充的明确零价优先于内嵌缺省值。
func TestByteSeekImageCacheSupplementPrecedence(t *testing.T) {
	s := newHotReloadCatalog(t, `{"gpt-image-2":{"cache_read_input_image_token_cost":0}}`)
	require.Zero(t, s.GetModelPricing("gpt-image-2").CacheReadInputImageTokenCost)
}
