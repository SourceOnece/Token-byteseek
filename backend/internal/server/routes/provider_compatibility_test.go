package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/handler"
	adminhandler "github.com/TokenFlux/TokenRouter/internal/handler/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Provider 入口先验证路由覆盖，响应 DTO/仓储迁移在后续阶段单独验收。
func TestProviderCompatibilityRoutesKeepAccountAndTicketEntrypoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api/v1/admin")
	h := &handler.Handlers{Admin: &handler.AdminHandlers{
		Account:          &adminhandler.AccountHandler{},
		Setting:          &adminhandler.SettingHandler{},
		OAuth:            &adminhandler.OAuthHandler{},
		OpenAIOAuth:      &adminhandler.OpenAIOAuthHandler{},
		CodexInviteReset: &adminhandler.CodexInviteResetHandler{},
	}}
	registerProviderCompatibilityRoutes(admin, h, func(c *gin.Context) { c.Next() })
	registerAccountRoutes(admin, h, func(c *gin.Context) { c.Next() })

	routes := map[string]bool{}
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
		if strings.HasPrefix(route.Path, "/api/v1/admin/accounts") {
			alias := strings.Replace(route.Path, "/accounts", "/providers", 1)
			matched := false
			for _, candidate := range router.Routes() {
				if candidate.Method == route.Method && candidate.Path == alias {
					require.Equal(t, route.Handler, candidate.Handler)
					matched = true
				}
			}
			require.True(t, matched, "missing alias %s %s", route.Method, alias)
		}
	}
	for _, route := range []string{
		http.MethodGet + " /api/v1/admin/providers",
		http.MethodGet + " /api/v1/admin/providers/:id",
		http.MethodPost + " /api/v1/admin/providers",
		http.MethodPut + " /api/v1/admin/providers/:id",
		http.MethodDelete + " /api/v1/admin/providers/:id",
		http.MethodPost + " /api/v1/admin/providers/codex-quality-test",
		http.MethodPost + " /api/v1/admin/providers/codex-ticket-collect",
		http.MethodGet + " /api/v1/admin/providers/:id/codex-ticket-settings",
		http.MethodGet + " /api/v1/admin/providers/codex-ticket-runs",
	} {
		require.True(t, routes[route], "missing compatibility route %s", route)
	}
}

// 管理员认证与导出二次验证在两个入口都必须早于真正的导出处理器。
func TestProviderCompatibilityExportAuthAndStepUp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, allowAdmin := range []bool{false, true} {
		router := gin.New()
		group := router.Group("/api/v1/admin", func(c *gin.Context) {
			if !allowAdmin {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			c.Next()
		})
		h := &handler.Handlers{Admin: &handler.AdminHandlers{}}
		stepCalls := 0
		step := func(c *gin.Context) { stepCalls++; c.AbortWithStatus(http.StatusForbidden) }
		registerProviderCompatibilityRoutes(group, h, step)
		registerAccountRoutes(group, h, step)
		for _, prefix := range []string{"accounts", "providers"} {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/"+prefix+"/data", nil))
			want := http.StatusUnauthorized
			if allowAdmin {
				want = http.StatusForbidden
			}
			require.Equal(t, want, w.Code)
		}
		if allowAdmin {
			require.Equal(t, 2, stepCalls)
		} else {
			require.Zero(t, stepCalls)
		}
	}
}
