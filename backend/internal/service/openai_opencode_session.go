package service

import (
	"crypto/sha256"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"net/http"
	"net/url"
	"strings"
)

const openCodeSessionHeader = "X-OpenCode-Session"

// 只把会话标识发送到官方 OpenCode 域名，避免把用户会话头泄露给自定义中继。
func applyOpenCodeSessionHeader(c *gin.Context, account *Account, targetURL string, headers http.Header, bodies ...[]byte) {
	if c == nil || c.Request == nil || account == nil || account.Type != AccountTypeAPIKey || headers == nil {
		return
	}
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return
	}
	if !account.IsOpenCodeGo() && (!strings.EqualFold(parsed.Scheme, "https") || !strings.EqualFold(parsed.Hostname(), "opencode.ai")) {
		return
	}
	value := strings.TrimSpace(c.GetHeader(openCodeSessionHeader))
	if account.IsOpenCodeGo() {
		value = openCodeScopedSession(c, account, bodies...)
	}
	if value == "" {
		return
	}
	for key := range headers {
		if strings.EqualFold(key, openCodeSessionHeader) {
			delete(headers, key)
		}
	}
	headers.Set(openCodeSessionHeader, value)
}

// 协议转换前仅保存会话候选字符串，不在 Gin Context 再保留大图片正文。
func rememberOpenCodeInboundSession(c *gin.Context, body []byte) {
	if c == nil {
		return
	}
	if _, exists := c.Get("opencode_original_session"); exists {
		return
	}
	c.Set("opencode_original_session", openCodeSessionFromBody(body))
}

func openCodeSessionFromBody(body []byte) string {
	if value := sanitizeSessionID(gjson.GetBytes(body, "prompt_cache_key").String()); value != "" {
		return value
	}
	metadata := gjson.GetBytes(body, "metadata.user_id").String()
	if parsed := ParseMetadataUserID(metadata); parsed != nil {
		return sanitizeSessionID(parsed.SessionID)
	}
	// 普通 user_id 并非会话 ID，不能把同一用户所有对话折叠到一个会话。
	return ""
}

func openCodeScopedSession(c *gin.Context, account *Account, bodies ...[]byte) string {
	if c == nil || account == nil {
		return ""
	}
	raw := sanitizeSessionID(c.GetHeader(openCodeSessionHeader))
	if raw == "" {
		raw = sanitizeSessionID(explicitOpenAIHeaderSessionID(c))
	}
	if raw == "" {
		raw = sanitizeSessionID(ClaudeCodeSessionIDFromHeader(c))
	}
	if raw == "" {
		raw = c.GetString("opencode_original_session")
	}
	for _, body := range bodies {
		if raw != "" {
			break
		}
		raw = openCodeSessionFromBody(body)
	}
	if raw == "" {
		raw = c.GetString("opencode_generated_session")
		if raw == "" {
			raw = uuid.NewString()
			c.Set("opencode_generated_session", raw)
		}
	}
	// 同 Key/账号/凭据/对话稳定，换 Key 或账号后隔离；摘要不包含明文凭据。
	credential := sha256.Sum256([]byte(account.GetCredential("api_key")))
	return deriveStableUUIDv4(fmt.Sprintf("byteseek:opencode:v1:key:%d:account:%d:credential:%x:session:%s", getAPIKeyIDFromContext(c), account.ID, credential, raw))
}
