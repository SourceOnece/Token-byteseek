package service

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strings"
)

const openCodeSessionHeader = "X-OpenCode-Session"

// 只把会话标识发送到官方 OpenCode 域名，避免把用户会话头泄露给自定义中继。
func applyOpenCodeSessionHeader(c *gin.Context, account *Account, targetURL string, headers http.Header) {
	if c == nil || c.Request == nil || account == nil || account.Type != AccountTypeAPIKey || headers == nil {
		return
	}
	parsed, err := url.Parse(targetURL)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || !strings.EqualFold(parsed.Hostname(), "opencode.ai") {
		return
	}
	value := strings.TrimSpace(c.GetHeader(openCodeSessionHeader))
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
