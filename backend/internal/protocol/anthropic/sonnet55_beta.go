package anthropic

import (
	"strings"

	"github.com/tidwall/gjson"
)

// FilterSonnet55ToolsetBeta 为直连与 Vertex 共享精确工具集规则，其他型号和 beta 保持原样。
func FilterSonnet55ToolsetBeta(header string, body []byte, model string) string {
	if i := strings.IndexByte(model, '@'); i >= 0 {
		model = model[:i]
	}
	if !IsSonnet55(model) {
		return header
	}
	for _, tool := range gjson.GetBytes(body, "tools").Array() {
		switch tool.Get("type").String() {
		case "computer_toolset_20260801", "browser_toolset_20260801":
			tokens := ParseAnthropicBetaHeader(header)
			out := tokens[:0]
			for _, token := range tokens {
				if token != "fine-grained-tool-streaming-2025-05-14" {
					out = append(out, token)
				}
			}
			return strings.Join(out, ",")
		}
	}
	return header
}
