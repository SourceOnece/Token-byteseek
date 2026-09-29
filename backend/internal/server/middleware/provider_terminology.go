package middleware

import (
	"bytes"
	"io"
	"strings"

	response "github.com/TokenFlux/TokenRouter/internal/server/httpx"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// ProviderTerminology 拒绝管理契约中的旧上游账号字段；凭据与第三方导入载荷交给所属适配器。
func ProviderTerminology() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/api/v1/admin/")
		// ByteSeek 私有扩展继续使用稳定的 account_ids，不受提供商名词改名影响。
		parts := strings.Split(path, "/")
		custom := path == "settings/codex-ticket" || strings.HasPrefix(path, "settings/codex-ticket/")
		if len(parts) > 1 && parts[0] == "providers" {
			custom = custom || strings.HasPrefix(parts[1], "codex-ticket-") || strings.HasPrefix(parts[1], "codex-quality-")
			custom = custom || len(parts) == 3 && parts[2] == "codex-ticket-settings"
		}
		if custom {
			c.Next()
			return
		}
		section, _, _ := strings.Cut(path, "/")
		switch section {
		case "providers", "groups", "pricing-configs", "proxies", "settings", "ops", "usage", "dashboard":
		default:
			c.Next()
			return
		}
		legacy := func(key string) bool {
			for _, part := range strings.Split(key, "_") {
				if part == "account" || part == "accounts" {
					return true
				}
			}
			return false
		}
		reject := func(key string) {
			response.BadRequest(c, "field "+key+" has been renamed to "+strings.ReplaceAll(key, "account", "provider"))
			c.Abort()
		}
		for key := range c.Request.URL.Query() {
			if legacy(key) {
				reject(key)
				return
			}
		}
		if c.ContentType() == "application/json" && c.Request.Body != nil && c.Request.ContentLength != 0 {
			var fields map[string]any
			if err := c.ShouldBindBodyWith(&fields, binding.JSON); err != nil {
				response.BadRequest(c, "Invalid request: "+err.Error())
				c.Abort()
				return
			}
			for key := range fields {
				if legacy(key) {
					reject(key)
					return
				}
			}
			// Gin 的缓存供 ShouldBindBodyWith 使用；普通 JSON 绑定仍需读取同一请求体。
			if raw, ok := c.Get(gin.BodyBytesKey); ok {
				if body, ok := raw.([]byte); ok {
					c.Request.Body = io.NopCloser(bytes.NewReader(body))
				}
			}
		}
		c.Next()
	}
}
