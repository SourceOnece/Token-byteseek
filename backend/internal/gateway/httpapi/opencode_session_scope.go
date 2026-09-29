package httpapi

import (
	"crypto/sha256"
	"fmt"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
	"github.com/TokenFlux/TokenRouter/internal/protocol/anthropic"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// 协议转换前只记住会话候选，不复制包含图片的请求正文。
func rememberOpenCodeInboundSession(c *gin.Context, body []byte) {
	if c == nil {
		return
	}
	if _, exists := c.Get("opencode_original_session"); exists {
		return
	}
	raw := session.SanitizeClientSessionID(gjson.GetBytes(body, "prompt_cache_key").String())
	if raw == "" {
		if parsed := anthropic.ParseMetadataUserID(gjson.GetBytes(body, "metadata.user_id").String()); parsed != nil {
			raw = session.SanitizeClientSessionID(parsed.SessionID)
		}
	}
	c.Set("opencode_original_session", raw)
}

// 相同 Key、账号、凭据和会话稳定；缺会话时仅在单次请求内复用随机身份。
func openCodeScopedSession(c *gin.Context, value *gatewayprovider.ExecutionProvider) string {
	raw := session.SanitizeClientSessionID(c.GetHeader(openCodeSessionHeader))
	if raw == "" {
		raw = session.SanitizeClientSessionID(ExplicitOpenAIHeaderSessionID(c))
	}
	if raw == "" {
		raw = session.SanitizeClientSessionID(ClaudeCodeSessionIDFromHeader(c))
	}
	if raw == "" {
		raw = c.GetString("opencode_original_session")
	}
	if raw == "" {
		raw = c.GetString("opencode_generated_session")
		if raw == "" {
			raw = uuid.NewString()
			c.Set("opencode_generated_session", raw)
		}
	}
	credential := sha256.Sum256([]byte(value.View().GetCredential("api_key")))
	return openai.DeriveStableUUIDv4(fmt.Sprintf("byteseek:opencode:v1:key:%d:account:%d:credential:%x:session:%s", APIKeyIDFromContext(c), value.Record.ID, credential, raw))
}
