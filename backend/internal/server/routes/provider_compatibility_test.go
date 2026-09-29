package routes

import (
	"net/http"
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
		Account: &adminhandler.AccountHandler{},
		Setting: &adminhandler.SettingHandler{},
	}}
	registerProviderCompatibilityRoutes(admin, h, func(c *gin.Context) { c.Next() })

	routes := map[string]bool{}
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
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
