package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
)

// 推理缓存只能回到同一租户Key、分组与账号身份；不信任客户端条目ID作为全局查找键。
// 缺少真实认证上下文时直接停用缓存，不能落入所有测试/匿名请求共享的命名空间。
// @project-doc docs/interfaces/openai_upstream.md#responses_reasoning_cache_scope
func responsesReasoningScope(c *gin.Context, account *Account) string {
	if c == nil || account == nil || account.ID <= 0 {
		return ""
	}
	value, _ := c.Get("api_key")
	key, ok := value.(*APIKey)
	if !ok || key == nil || key.ID <= 0 || key.UserID <= 0 {
		return ""
	}
	raw, err := json.Marshal([]any{key.UserID, key.ID, key.TeamID, key.GroupID, account.ID, account.Platform, account.Type, account.Credentials})
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

func scopedResponsesReasoningKey(scope, itemID string) string {
	if scope == "" || strings.TrimSpace(itemID) == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(scope + "\x00" + itemID))
	return "scoped-v2:" + hex.EncodeToString(hash[:])
}

// 每次请求创建独立闭包，不把账号身份存到共享GatewayService字段。
func (s *OpenAIGatewayService) reasoningResolver(scope string) func(string) string {
	return func(itemID string) string { return s.reasoningContentByID(scope, itemID) }
}
