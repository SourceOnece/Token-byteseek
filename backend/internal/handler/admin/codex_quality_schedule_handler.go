package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/TokenFlux/TokenRouter/internal/pkg/response"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *AccountHandler) ListQualitySchedules(c *gin.Context) {
	repo, ok := h.accountTestService.QualityScheduleRepository()
	if !ok {
		response.Error(c, 503, "定时检测不可用")
		return
	}
	plans, err := repo.ListQualitySchedules(c.Request.Context())
	if err != nil {
		response.Error(c, 500, "读取计划失败")
		return
	}
	response.Success(c, plans)
}
func (h *AccountHandler) SaveQualitySchedule(c *gin.Context) {
	var plan service.CodexQualitySchedule
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128*1024)
	if c.ShouldBindJSON(&plan) != nil {
		response.BadRequest(c, "计划参数无效")
		return
	}
	plan.ID = 0
	if c.Param("id") != "" {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "计划 ID 无效")
			return
		}
		plan.ID = id
	}
	if err := plan.Validate(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	// 保存时验证固定账号集合，后续新导入账号不会自动加入。
	if err := h.accountTestService.ValidateQualityScheduleAccounts(c.Request.Context(), plan.Config.AccountIDs); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	repo, ok := h.accountTestService.QualityScheduleRepository()
	if !ok {
		response.Error(c, 503, "定时检测不可用")
		return
	}
	if err := repo.SaveQualitySchedule(c.Request.Context(), &plan); err != nil {
		response.Error(c, 500, "保存计划失败")
		return
	}
	response.Success(c, plan)
}
func (h *AccountHandler) SetQualityScheduleEnabled(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "计划 ID 无效")
		return
	}
	var req struct {
		Enabled *bool `json:"enabled"`
		Confirm bool  `json:"confirm_scheduling"`
	}
	if c.ShouldBindJSON(&req) != nil || req.Enabled == nil || (*req.Enabled && !req.Confirm) {
		response.BadRequest(c, "启用前必须确认调度影响")
		return
	}
	repo, ok := h.accountTestService.QualityScheduleRepository()
	if !ok {
		response.Error(c, 503, "定时检测不可用")
		return
	}
	if err = repo.SetQualityScheduleEnabled(c.Request.Context(), id, *req.Enabled); err != nil {
		response.Error(c, 400, "计划不存在或更新失败")
		return
	}
	response.Success(c, gin.H{"updated": true})
}

// TriggerQualitySchedule 请求 runner 尽快领取一次，不在 HTTP 请求中执行上游调用。
func (h *AccountHandler) TriggerQualitySchedule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "计划 ID 无效")
		return
	}
	repo, ok := h.accountTestService.QualityScheduleRepository()
	if !ok {
		response.Error(c, 503, "定时检测不可用")
		return
	}
	if err := repo.TriggerQualitySchedule(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrQualityScheduleBusy) {
			response.Error(c, 409, err.Error())
		} else {
			response.Error(c, 500, "发起检测失败")
		}
		return
	}
	response.Success(c, gin.H{"triggered": true})
}
func (h *AccountHandler) ListQualityRuns(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "计划 ID 无效")
		return
	}
	repo, ok := h.accountTestService.QualityScheduleRepository()
	if !ok {
		response.Error(c, 503, "定时检测不可用")
		return
	}
	runs, err := repo.ListQualityRuns(c.Request.Context(), id)
	if err != nil {
		response.Error(c, 500, "读取历史失败")
		return
	}
	response.Success(c, runs)
}
func (h *AccountHandler) QualityRunDetail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "轮次 ID 无效")
		return
	}
	status := c.Query("status")
	if !service.ValidCodexQualityFilter(status) || status == "untested" {
		response.BadRequest(c, "状态无效")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	repo, ok := h.accountTestService.QualityScheduleRepository()
	if !ok {
		response.Error(c, 503, "定时检测不可用")
		return
	}
	run, err := repo.GetQualityRun(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "轮次不存在")
		return
	}
	results, total, err := repo.ListQualityRunResults(c.Request.Context(), id, status, page, 20)
	if err != nil {
		response.Error(c, 500, "读取结果失败")
		return
	}
	response.Success(c, gin.H{"run": run, "items": results, "total": total, "page": page, "page_size": 20})
}
