package httpapi

import (
	"github.com/TokenFlux/TokenRouter/internal/quality"
	"github.com/gin-gonic/gin"
)

type Handler struct{ accountTestService *quality.Service }

func New(core *quality.Service) *Handler { return &Handler{accountTestService: core} }

// 两种管理名称共用同一个检测服务，保留既有客户端接口。
func Register(admin *gin.RouterGroup, h *Handler) {
	for _, prefix := range []string{"/accounts", "/providers"} {
		group := admin.Group(prefix)
		group.POST("/codex-quality-test", h.BatchCodexQualityTest)
		group.GET("/codex-quality-results", h.ListCodexQualityResults)
		group.GET("/codex-quality-stats", h.CodexQualityStats)
		group.GET("/codex-quality-schedules", h.ListQualitySchedules)
		group.POST("/codex-quality-schedules", h.SaveQualitySchedule)
		group.PUT("/codex-quality-schedules/:id", h.SaveQualitySchedule)
		group.DELETE("/codex-quality-schedules/:id", h.DeleteQualitySchedule)
		group.PUT("/codex-quality-schedules/:id/enabled", h.SetQualityScheduleEnabled)
		group.POST("/codex-quality-schedules/:id/run", h.TriggerQualitySchedule)
		group.GET("/codex-quality-schedules/:id/runs", h.ListQualityRuns)
		group.GET("/codex-quality-runs/:id", h.QualityRunDetail)
	}
}
