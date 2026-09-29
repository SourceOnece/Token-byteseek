package httpapi

import (
	"github.com/TokenFlux/TokenRouter/internal/codexticket"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	codexTickets *codexticket.CodexTicketService
}

func New(core *codexticket.CodexTicketService) *Handler { return &Handler{codexTickets: core} }

// 私有配置和历史仅挂载管理员组，代理密文和 STATE 不进入公开设置。
func Register(admin *gin.RouterGroup, h *Handler) {
	settings := admin.Group("/settings/codex-ticket")
	settings.GET("", h.GetCodexTicketSettings)
	settings.PUT("", h.UpdateCodexTicketSettings)
	settings.GET("/status", h.GetCodexTicketStatus)
	settings.POST("/proxy-test", h.TestCodexTicketProxy)
	for _, prefix := range []string{"/accounts", "/providers"} {
		group := admin.Group(prefix)
		group.GET("/codex-ticket-import-defaults", h.GetCodexTicketImportDefaults)
		group.PUT("/codex-ticket-import-defaults", h.UpdateCodexTicketImportDefaults)
		group.POST("/codex-ticket-collect", h.BatchCodexTicketCollect)
		group.GET("/:id/codex-ticket-settings", h.GetCodexTicketAccountSettings)
		group.PUT("/codex-ticket-settings", h.UpdateCodexTicketAccountSettings)
		group.GET("/codex-ticket-runs", h.ListCodexTicketRuns)
		group.GET("/codex-ticket-runs/:id", h.CodexTicketRunDetail)
		group.DELETE("/codex-ticket-runs", h.DeleteCodexTicketHistory)
		group.DELETE("/codex-ticket-runs/:id", h.DeleteCodexTicketHistory)
		group.DELETE("/codex-ticket-runs/:id/events/:event_id", h.DeleteCodexTicketHistory)
	}
}
