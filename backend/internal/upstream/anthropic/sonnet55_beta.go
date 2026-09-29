package anthropic

import (
	claudewire "github.com/TokenFlux/TokenRouter/internal/protocol/anthropic"
	"github.com/tidwall/gjson"
	"net/http"
	"strings"
)

// Sonnet 5.5 的新工具集不能同时带旧工具流 beta，其他 beta 原样保留。
func FilterSonnet55ToolsetBeta(header string, body []byte, model string) string {
	if i := strings.IndexByte(model, '@'); i >= 0 {
		model = model[:i]
	}
	if !claudewire.IsSonnet55(model) {
		return header
	}
	for _, tool := range gjson.GetBytes(body, "tools").Array() {
		switch tool.Get("type").String() {
		case "computer_toolset_20260801", "browser_toolset_20260801":
			return StripBetaTokensWithSet(header, map[string]struct{}{BetaFineGrainedToolStreaming: {}})
		}
	}
	return header
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
