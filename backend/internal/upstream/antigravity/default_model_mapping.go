package antigravity

// AntigravityGemini31ProAgentModel 是 Gemini 3.1 Pro High 的实际上游路由模型。
const AntigravityGemini31ProAgentModel = "gemini-pro-agent"

// DefaultAntigravityModelMapping 是 Antigravity 平台的默认模型映射
// 当提供商未配置 model_mapping 时使用此默认值
// 前端通过管理接口读取同一目录，显式提供商映射不会被本表补写。
var DefaultAntigravityModelMapping = map[string]string{
	// Claude 白名单
	"claude-fable-5-1":           "claude-fable-5-1",         // 官方模型
	"claude-fable-5":             "claude-fable-5",           // 官方模型
	"claude-opus-4-8":            "claude-opus-4-8",          // 官方模型
	"claude-opus-4-7":            "claude-opus-4-7",          // 官方模型
	"claude-opus-4-6-thinking":   "claude-opus-4-6-thinking", // 官方模型
	"claude-sonnet-4-6":          "claude-sonnet-4-6",
	"claude-sonnet-4-5":          "claude-sonnet-4-5",
	"claude-sonnet-4-5-thinking": "claude-sonnet-4-5-thinking",
	// Gemini 2.5 白名单
	"gemini-2.5-flash":          "gemini-2.5-flash",
	"gemini-2.5-flash-image":    "gemini-2.5-flash-image",
	"gemini-2.5-flash-lite":     "gemini-2.5-flash-lite",
	"gemini-2.5-flash-thinking": "gemini-2.5-flash-thinking",
	"gemini-2.5-pro":            "gemini-2.5-pro",
	// Gemini 3 白名单
	"gemini-3-flash":    "gemini-3-flash",
	"gemini-3-pro-high": "gemini-3-pro-high",
	"gemini-3-pro-low":  "gemini-3-pro-low",
	// Gemini 3.1 白名单
	AntigravityGemini31ProAgentModel: AntigravityGemini31ProAgentModel,
	"gemini-3.1-pro-high":            AntigravityGemini31ProAgentModel,
	"gemini-3.1-pro-low":             "gemini-3.1-pro-low",
	// Gemini 3.1 image 白名单
	"gemini-3.1-flash-image": "gemini-3.1-flash-image",
	// Gemini 3.6 Flash 分档模型
	"gemini-3.6-flash":        "gemini-3.6-flash",
	"gemini-3.6-flash-high":   "gemini-3.6-flash-high",
	"gemini-3.6-flash-low":    "gemini-3.6-flash-low",
	"gemini-3.6-flash-medium": "gemini-3.6-flash-medium",
	"gemini-3.6-flash-tiered": "gemini-3.6-flash-tiered",
	// 其他官方模型
	"gpt-oss-120b-medium":    "gpt-oss-120b-medium",
	"tab_flash_lite_preview": "tab_flash_lite_preview",
}
