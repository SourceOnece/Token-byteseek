package antigravity

import (
	"github.com/TokenFlux/TokenRouter/internal/protocol/bridge"
	"strings"
)

// 有客户端函数时不能将整个请求降为纯服务端搜索。
func hasClientFunctionTools(tools []ClaudeTool) bool {
	for _, tool := range tools {
		if bridge.InternalIsWebSearchTool(tool) || bridge.InternalIsCodeExecutionTool(tool) {
			continue
		}
		if tool.Type == "custom" && (tool.Custom == nil || tool.Custom.InputSchema == nil) {
			continue
		}
		if strings.TrimSpace(tool.Name) != "" {
			return true
		}
	}
	return false
}
