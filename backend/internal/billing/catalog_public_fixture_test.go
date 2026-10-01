package billing_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 普通构建也验证发布资源入口，确保目录迁移没有丢失 fork 的精确型号补充。
func TestByteSeekPublishedCatalogModels(t *testing.T) {
	catalog := newOfflineCatalogFixture(t)
	for _, model := range []string{"gpt-6.1-sol", "claude-sonnet-5-5", "claude-opus-5-5", "gpt-image-2.5-flare"} {
		price := catalog.GetModelPricing(model)
		require.NotNil(t, price, model)
		require.NotEqual(t, "unpriced", price.Source, model)
	}
	price := catalog.GetModelPricing("gpt-image-2")
	require.NotNil(t, price)
	require.InDelta(t, 2e-6, price.CacheReadInputImageTokenCost, 1e-12)
}
