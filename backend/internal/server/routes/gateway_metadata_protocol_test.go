package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/server/middleware"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 三类协议八种组合与账号开关两态：Metadata 不能绕过协议门禁，也不反向关闭协议。
func TestGatewayMetadataRepairDoesNotChangeGroupProtocols(t *testing.T) {
	gin.SetMode(gin.TestMode)
	protocols := []service.GroupClientProtocol{service.GroupClientProtocolAnthropicMessages, service.GroupClientProtocolOpenAIResponses, service.GroupClientProtocolOpenAIChatCompletions}
	paths := []string{"/v1/messages", "/v1/responses", "/v1/chat/completions"}
	for mask := 0; mask < 8; mask++ {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("mask=%d/repair=%v", mask, enabled), func(t *testing.T) {
				group := &service.Group{Platform: service.PlatformOpenAI}
				for i, p := range protocols {
					if mask&(1<<i) != 0 {
						group.AllowedClientProtocols = append(group.AllowedClientProtocols, p)
					}
				}
				router := gin.New()
				account := &service.Account{Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Extra: map[string]any{"codex_metadata_repair_enabled": enabled}}
				router.Use(func(c *gin.Context) {
					c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{ID: 101, Group: group})
					c.Next()
				})
				for i, p := range protocols {
					format := groupClientProtocolErrorOpenAI
					if i == 0 {
						format = groupClientProtocolErrorAnthropic
					}
					router.POST(paths[i], requireGroupClientProtocol(p, format), func(c *gin.Context) {
						require.Equal(t, enabled, account.IsCodexMetadataRepairEnabled())
						c.Status(http.StatusNoContent)
					})
				}
				for i, path := range paths {
					reader := &protocolGateTrackingReader{}
					w := httptest.NewRecorder()
					router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, reader))
					if mask&(1<<i) != 0 {
						require.Equal(t, http.StatusNoContent, w.Code)
					} else {
						require.Equal(t, http.StatusForbidden, w.Code)
					}
					require.False(t, reader.read)
				}
			})
		}
	}
}
