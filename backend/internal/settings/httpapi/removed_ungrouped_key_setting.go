package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/server/httpx"
	"github.com/gin-gonic/gin"
)

// 旧开关必须明确拒绝，避免管理员误以为未选组的 Key 仍可调用。
func rejectRemovedUngroupedKeySchedulingField(c *gin.Context, fields map[string]json.RawMessage) bool {
	const field = "allow_ungrouped_key_scheduling"
	if _, present := fields[field]; !present {
		return false
	}
	httpx.ErrorWithDetails(c, http.StatusBadRequest, "API keys require an explicit group", "REMOVED_SETTING_FIELD", map[string]string{"field": field})
	return true
}
