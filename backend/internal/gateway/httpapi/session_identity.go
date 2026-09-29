package httpapi

import (
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/gateway/requeststate"

	protocolopenai "github.com/TokenFlux/TokenRouter/internal/protocol/openai"

	"github.com/TokenFlux/TokenRouter/internal/scheduler"

	gatewaysession "github.com/TokenFlux/TokenRouter/internal/gateway/session"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"

	"github.com/TokenFlux/TokenRouter/internal/apikey"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
)

const (
	OpenCodeSessionAffinityHeader = "X-Session-Affinity"
	OpenCodeSessionIDHeader       = "X-Session-Id"
	OpenCodeNativeSessionHeader   = "X-OpenCode-Session"
	CodeBuddyConversationHeader   = "X-Conversation-ID"
)

var explicitOpenAIHeaderSessionNames = []string{
	"session-id",
	"session_id",
	"conversation_id",
	OpenCodeSessionAffinityHeader,
	OpenCodeSessionIDHeader,
	OpenCodeNativeSessionHeader,
	CodeBuddyConversationHeader,
}

const (
	GrokConversationIDHeader = "X-Grok-Conv-Id"
	ClaudeCodeSessionHeader  = "X-Claude-Code-Session-Id"
)

var clientSessionIDHeaders = append(append([]string(nil), explicitOpenAIHeaderSessionNames...), ClaudeCodeSessionHeader)

// ExplicitOpenAIHeaderSessionID 提取 OpenAI 兼容客户端发送的稳定会话标识。
// 这里只接收会话级字段；每轮变化的请求或消息 ID 会破坏粘性路由和上游提示缓存。
func ExplicitOpenAIHeaderSessionID(c *gin.Context) string {
	if c == nil {
		return ""
	}

	for _, header := range explicitOpenAIHeaderSessionNames {
		if sessionID := strings.TrimSpace(c.GetHeader(header)); sessionID != "" {
			return sessionID
		}
	}
	return ""
}

func ExplicitOpenAISessionID(c *gin.Context, body []byte) string {
	if c == nil {
		return ""
	}

	sessionID := ExplicitOpenAIHeaderSessionID(c)
	if sessionID == "" && len(body) > 0 {
		// WS response.create 将会话字段包在 response 内，先解包再读取统一信号。
		sessionID = strings.TrimSpace(protocolopenai.RequestPayloadView(body).Get("prompt_cache_key").String())
	}
	return sessionID
}

// ExplicitOpenAIRequestSessionID 按客户端显式会话信号读取身份，选定提供商后再应用其平台约束。
func ExplicitOpenAIRequestSessionID(c *gin.Context, body []byte) string {
	if c == nil {
		return ""
	}

	sessionID := ExplicitOpenAIHeaderSessionID(c)
	if sessionID == "" && IsGrokRequestContext(c) {
		sessionID = strings.TrimSpace(c.GetHeader(GrokConversationIDHeader))
	}
	if sessionID == "" && len(body) > 0 {
		sessionID = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	}
	if sessionID == "" && IsGrokRequestContext(c) && len(body) > 0 {
		sessionID = gatewaysession.GrokPreviousResponseSeed(body)
	}
	return sessionID
}

// GenerateExplicitOpenAISessionHash 只采用显式会话信号，图片等无状态入口不使用内容回退。
func GenerateExplicitOpenAISessionHash(c *gin.Context, body []byte) string {
	sessionID := ExplicitOpenAIRequestSessionID(c, body)
	if sessionID == "" {
		return ""
	}

	currentHash, legacyHash := scheduler.DeriveSessionHashes(sessionID)
	AttachOpenAILegacySessionHash(c, legacyHash)
	return currentHash
}

// GenerateOpenAISessionHash 为 OpenAI 请求生成粘性会话哈希。
// 优先级依次为：session-id/session_id、conversation_id、OpenCode 会话头、
// CodeBuddy 会话头、Grok 原生会话头、prompt_cache_key，最后才使用内容回退。
func GenerateOpenAISessionHash(c *gin.Context, body []byte) string {
	if c == nil {
		return ""
	}

	sessionID := ExplicitOpenAIRequestSessionID(c, body)
	if sessionID == "" && len(body) > 0 {
		sessionID = gatewaysession.OpenAIContentSeed(body)
	}
	if sessionID == "" {
		return ""
	}

	if IsGrokRequestContext(c) {
		sessionID = gatewaysession.GrokStickyAffinitySeed(sessionID, body)
	}

	currentHash, legacyHash := scheduler.DeriveSessionHashes(sessionID)
	AttachOpenAILegacySessionHash(c, legacyHash)
	return currentHash
}

// GenerateOpenAISessionHashWithFallback 先按常规信号生成会话哈希；
// 当未携带 session_id/conversation_id/prompt_cache_key 时，使用 fallbackSeed 生成稳定哈希。
// 该方法用于 WS ingress，避免会话信号缺失时发生跨提供商漂移。
func GenerateOpenAISessionHashWithFallback(c *gin.Context, body []byte, fallbackSeed string) string {
	sessionHash := GenerateOpenAISessionHash(c, body)
	if sessionHash != "" {
		return sessionHash
	}

	seed := strings.TrimSpace(fallbackSeed)
	if seed == "" {
		return ""
	}

	currentHash, legacyHash := scheduler.DeriveSessionHashes(seed)
	AttachOpenAILegacySessionHash(c, legacyHash)
	return currentHash
}

// ClaudeCodeSessionIDFromHeader 解析 Claude Code 会话头，用于消息协议的粘性路由。
// 该入口与仅用于用量日志的 ExtractClientSessionID 分离，避免改变其他协议的会话语义。
func ClaudeCodeSessionIDFromHeader(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return gatewaysession.SanitizeClientSessionID(c.GetHeader(ClaudeCodeSessionHeader))
}

func ExtractClientSessionID(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return gatewaysession.ExtractClientSessionID(c.GetHeader, clientSessionIDHeaders, IsGrokRequestContext(c))
}

// IsGrokRequestContext 使用已选提供商、强制路由或显式 Grok 会话头，不推断分组平台。
func IsGrokRequestContext(c *gin.Context) bool {
	if c == nil {
		return false
	}
	if platform := selectedOpsPlatform(c); platform != "" && platform != "unknown" {
		return platform == capability.PlatformGrok
	}
	if c.Request != nil {
		if platform, ok := apikey.ForcePlatformFromContext(c.Request.Context()); ok && strings.TrimSpace(platform) != "" {
			return platform == capability.PlatformGrok
		}
	}
	return strings.TrimSpace(c.GetHeader(GrokConversationIDHeader)) != ""
}

func AttachOpenAILegacySessionHash(c *gin.Context, legacyHash string) {
	if c == nil || c.Request == nil {
		return
	}
	c.Request = c.Request.WithContext(requeststate.WithOpenAILegacySessionHash(c.Request.Context(), legacyHash))
}
