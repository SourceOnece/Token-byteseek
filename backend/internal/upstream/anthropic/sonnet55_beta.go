package anthropic

import (
	"net/http"

	claudewire "github.com/TokenFlux/TokenRouter/internal/protocol/anthropic"
)

// Sonnet 5.5 的新工具集不能同时带旧工具流 beta，其他 beta 原样保留。
func FilterSonnet55ToolsetBeta(header string, body []byte, model string) string {
	return claudewire.FilterSonnet55ToolsetBeta(header, body, model)
}

// 覆写请求头后再核对一次，避免配置重新注入不兼容的 beta。
func FilterSonnet55ToolsetBetaHeader(header http.Header, body []byte, model string) {
	original := GetHeaderRaw(header, "anthropic-beta")
	filtered := FilterSonnet55ToolsetBeta(original, body, model)
	if filtered == original {
		return
	}
	DeleteHeaderAllForms(header, "anthropic-beta")
	if filtered != "" {
		SetHeaderRaw(header, "anthropic-beta", filtered)
	}
}
