package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 已删除的额度接口不再注册到用户和管理员路由。
func TestRemovedPlatformQuotaRoutesReturnNotFound(t *testing.T) {
	router := gin.New()
	RegisterUserRoutes(router.Group("/api/v1"), &RedeemHandler{}, &SubscriptionHandler{})
	RegisterSubscriptionRoutes(router.Group("/api/v1/admin"), &AdminSubscriptionHandler{})
	for _, item := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/user/platform-quotas"},
		{http.MethodGet, "/api/v1/admin/users/1/platform-quotas"},
		{http.MethodPut, "/api/v1/admin/users/1/platform-quotas"},
		{http.MethodPost, "/api/v1/admin/users/1/platform-quotas/reset"},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(item.method, item.path, nil))
		require.Equal(t, http.StatusNotFound, recorder.Code, "%s %s", item.method, item.path)
	}
}
