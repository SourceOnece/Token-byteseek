package completion

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/stretchr/testify/require"
)

// 实际输出尺寸覆盖请求尺寸，并进入生产完成处理使用的计费元数据。
func TestNormalizeResultUsesDecodedImageDimensions(t *testing.T) {
	result := &Result{ImageCount: 1, ImageInputSize: "3840x2160", ImageOutputSizes: []string{"1672x941"}}
	(&Recorder{}).normalizeResult(result, &ProviderSnapshot{OAuthLike: true}, true, nil)
	require.Equal(t, pricing.ImageBillingSize2K, result.ImageSize)
	require.Equal(t, "1672x941", result.ImageOutputSize)
	require.Equal(t, pricing.ImageSizeSourceOutput, result.ImageSizeSource)
}
