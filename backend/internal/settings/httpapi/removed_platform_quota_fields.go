package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	response "github.com/TokenFlux/TokenRouter/internal/server/httpx"
	"github.com/gin-gonic/gin"
)

// rejectRemovedPlatformQuotaFields 拒绝已移除的设置，避免旧客户端误以为限额仍然生效。
func rejectRemovedPlatformQuotaFields(c *gin.Context, fields map[string]json.RawMessage) bool {
	for name := range fields {
		if name == "default_platform_quotas" || strings.HasPrefix(name, "auth_source_default_") && strings.HasSuffix(name, "_platform_quotas") {
			response.ErrorWithDetails(c, http.StatusBadRequest, "User platform quotas have been removed", "REMOVED_SETTING_FIELD", map[string]string{"field": name})
			return true
		}
	}
	return false
}
