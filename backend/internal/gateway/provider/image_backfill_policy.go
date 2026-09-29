package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
)

// ProviderExtraImagesURLToB64JSON 保留既有提供商设置键。
const ProviderExtraImagesURLToB64JSON = "images_url_to_b64_json"

// ImagesURLToB64JSONEnabled 返回提供商是否开启 URL 到 base64 的图片回填。
func ImagesURLToB64JSONEnabled(provider *ExecutionProvider) bool {
	return provider != nil && provider.Record.Platform == capability.PlatformOpenAI && provider.Record.Type == capability.ProviderTypeAPIKey && ExecutionProtocolRecord(provider).GetExtraBool(ProviderExtraImagesURLToB64JSON)
}
