package capability

// 平台与提供商类型属于能力目录，不依赖旧业务实体。
// Platform constants
const (
	PlatformAnthropic   = "anthropic"
	PlatformOpenAI      = "openai"
	PlatformGemini      = "gemini"
	PlatformAntigravity = "antigravity"
	PlatformGrok        = "grok"
	PlatformQoder       = "qoder"
	PlatformKimi        = "kimi"
	PlatformZhipu       = "zhipu"
	PlatformDeepseek    = "deepseek"
	PlatformMiniMax     = "minimax"
	PlatformOpenCodeGo  = "opencode_go"
)

// Provider type constants
const (
	ProviderTypeOAuth          = "oauth"           // OAuth类型提供商（full scope: profile + inference）
	ProviderTypeSetupToken     = "setup-token"     // Setup Token类型提供商（inference only scope）
	ProviderTypeAPIKey         = "apikey"          // API Key类型提供商
	ProviderTypeUpstream       = "upstream"        // 上游透传类型提供商（通过 Base URL + API Key 连接上游）
	ProviderTypeBedrock        = "bedrock"         // AWS Bedrock 类型提供商（通过 SigV4 签名或 API Key 连接 Bedrock，由 credentials.auth_mode 区分）
	ProviderTypeServiceAccount = "service_account" // Google Service Account 类型提供商（用于 Vertex AI）
	ProviderTypeCosy           = "cosy"            // Qoder COSY 协议提供商
)

const (
	OpenAIAuthModePersonalAccessToken = "personalAccessToken"
	OpenAIAuthModeAgentIdentity       = "agentIdentity"
)
