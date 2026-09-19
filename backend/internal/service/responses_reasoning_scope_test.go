package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 同名条目不能跨Key、账号、凭据、分组或团队读写；无认证上下文与旧裸ID缓存不回退。
func TestResponsesReasoningCacheTenantAndAccountIsolation(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	a := rawChatCompletionsTestAccount()
	group, team := int64(3), int64(4)
	key := &APIKey{ID: 1, UserID: 2, GroupID: &group, TeamID: &team}
	c.Set("api_key", key)
	cache := &reasoningHitCache{values: map[string]string{"same-item": "legacy must not be read"}}
	s := &OpenAIGatewayService{cache: cache}
	scope := responsesReasoningScope(c, a)
	require.NotEmpty(t, scope)
	require.Empty(t, s.reasoningContentByID(scope, "same-item"))
	s.setReasoningContent(scope, "same-item", "first tenant content")
	require.Equal(t, "first tenant content", s.reasoningContentByID(scope, "same-item"))
	for _, change := range []func(*APIKey, *Account){
		func(k *APIKey, _ *Account) { k.ID++ },
		func(k *APIKey, _ *Account) { k.UserID++ },
		func(k *APIKey, _ *Account) { k.TeamID = nil },
		func(k *APIKey, _ *Account) { k.GroupID = nil },
		func(_ *APIKey, a *Account) { a.ID++ },
		func(_ *APIKey, a *Account) { a.Credentials = map[string]any{"api_key": "different-synthetic"} },
	} {
		otherKey, otherAccount := *key, *a
		change(&otherKey, &otherAccount)
		c.Set("api_key", &otherKey)
		otherScope := responsesReasoningScope(c, &otherAccount)
		require.NotEqual(t, scope, otherScope)
		require.Empty(t, s.reasoningContentByID(otherScope, "same-item"))
		s.setReasoningContent(otherScope, "same-item", "other tenant")
		require.Equal(t, "first tenant content", s.reasoningContentByID(scope, "same-item"))
	}
	require.Empty(t, responsesReasoningScope(nil, a))
	require.Empty(t, s.reasoningContentByID("", "same-item"))
	s.setReasoningContent("", "same-item", "anonymous")
	require.Equal(t, "legacy must not be read", cache.values["same-item"])
}
