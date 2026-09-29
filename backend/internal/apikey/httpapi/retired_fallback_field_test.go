package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/identity/httpapi/authctx"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 身份认证后、读取 Key 服务前拒绝旧回退字段，包括显式 false 和 null。
func TestKeyWritesRejectRetiredDefaultGroupFallback(t *testing.T) {
	handler := NewAPIKeyHandler[struct{}](nil, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(authctx.ContextKeyUser, authctx.AuthSubject{UserID: 1}); c.Next() })
	router.POST("/keys", handler.Create)
	router.PUT("/keys/:id", handler.Update)
	for _, value := range []string{"true", "false", "null"} {
		for _, endpoint := range []struct{ method, path string }{{http.MethodPost, "/keys"}, {http.MethodPut, "/keys/1"}} {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader(`{"name":"key","group_id":1,"fallback_to_default_group_when_unavailable":`+value+`}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), "fallback_to_default_group_when_unavailable")
		}
	}
}
