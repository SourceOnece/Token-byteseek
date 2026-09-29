package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	keyhttp "github.com/TokenFlux/TokenRouter/internal/apikey/httpapi"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/gin-gonic/gin"
)

// 未认证时停用回查；凭据仅参与哈希，缓存键不保存其原文。
func responsesReasoningScope(c *gin.Context, value *gatewayprovider.ExecutionProvider) string {
	if c == nil || value == nil || value.Record.ID <= 0 {
		return ""
	}
	key, ok := keyhttp.GetAPIKeyFromContext(c)
	if !ok || key == nil || key.ID <= 0 || key.UserID <= 0 {
		return ""
	}
	a := value.View()
	raw, err := json.Marshal([]any{key.UserID, key.ID, key.TeamID, key.GroupID, a.ID, a.Platform, a.Type, a.Credentials})
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}
