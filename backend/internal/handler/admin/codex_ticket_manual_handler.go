package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/pkg/response"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 管理员显式确认后才开始消耗额度；SSE 断开取消剩余任务，已写历史可再次查询。
func (h *SettingHandler) BatchCodexTicketCollect(c *gin.Context) {
	if h.codexTickets == nil {
		response.Error(c, 503, "票据服务不可用")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32*1024)
	var req service.CodexTicketManualRequest
	if c.ShouldBindJSON(&req) != nil {
		response.BadRequest(c, "无效的批量采集参数")
		return
	}
	if err := req.Normalize(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	m, err := h.codexTickets.PrepareManualCollection(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 409, err.Error())
		return
	}
	defer m.Close()
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.Flush()
	m.Execute(func(kind string, data any) bool {
		// 长批次不设总时限，但单次向断流浏览器写入必须有界，防止持锁挂住。
		_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(15 * time.Second))
		encoded, err := json.Marshal(map[string]any{"type": kind, "data": data})
		if err != nil {
			return false
		}
		if _, err = fmt.Fprintf(c.Writer, "data: %s\n\n", encoded); err != nil {
			return false
		}
		c.Writer.Flush()
		return true
	})
}

// 用户明确要求直接删除，不设额外确认参数；仅管理员 DELETE 路由可调用。
func (h *SettingHandler) DeleteCodexTicketHistory(c *gin.Context) {
	runID := c.Param("id")
	if runID != "" {
		if _, err := uuid.Parse(runID); err != nil {
			response.BadRequest(c, "批次 ID 无效")
			return
		}
	}
	var eventID int64
	if value := c.Param("event_id"); value != "" {
		var err error
		eventID, err = strconv.ParseInt(value, 10, 64)
		if err != nil || eventID <= 0 {
			response.BadRequest(c, "日志 ID 无效")
			return
		}
	}
	deleted, err := h.codexTickets.DeleteTicketHistory(c.Request.Context(), runID, eventID)
	if errors.Is(err, service.ErrTicketHistoryActive) {
		response.Error(c, 409, err.Error())
		return
	}
	if errors.Is(err, service.ErrTicketHistoryNotFound) {
		response.Error(c, 404, err.Error())
		return
	}
	if err != nil {
		response.InternalError(c, "删除采集历史失败")
		return
	}
	response.Success(c, map[string]any{"deleted": deleted})
}
func (h *SettingHandler) ListCodexTicketRuns(c *gin.Context) {
	r, err := h.codexTickets.TicketHistory()
	if err != nil {
		response.Error(c, 503, err.Error())
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 || page > 1000000 {
		response.BadRequest(c, "页码无效")
		return
	}
	runs, err := r.ListTicketRuns(c.Request.Context(), page)
	if err != nil {
		response.InternalError(c, "读取采集批次失败")
		return
	}
	response.Success(c, runs)
}
func (h *SettingHandler) CodexTicketRunDetail(c *gin.Context) {
	r, err := h.codexTickets.TicketHistory()
	if err != nil {
		response.Error(c, 503, err.Error())
		return
	}
	id := c.Param("id")
	if _, err = uuid.Parse(id); err != nil {
		response.BadRequest(c, "批次 ID 无效")
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 || page > 1000000 {
		response.BadRequest(c, "页码无效")
		return
	}
	accountID, err := strconv.ParseInt(c.DefaultQuery("account_id", "0"), 10, 64)
	if err != nil || accountID < 0 {
		response.BadRequest(c, "账号 ID 无效")
		return
	}
	kind, status, model := c.DefaultQuery("kind", "result"), c.Query("status"), c.Query("model")
	if kind != "result" && kind != "attempt" {
		response.BadRequest(c, "日志类型无效")
		return
	}
	if model != "" && model != "gpt-6-astra" && model != "gpt-5.6-sol" {
		response.BadRequest(c, "模型无效")
		return
	}
	if status != "" && status != "ready" && status != "missing" && status != "failed" && status != "skipped" && status != "cancelled" {
		response.BadRequest(c, "状态无效")
		return
	}
	run, err := r.GetTicketRun(c.Request.Context(), id)
	if err != nil {
		response.Error(c, 404, "批次不可用")
		return
	}
	items, total, err := r.ListTicketEvents(c.Request.Context(), id, kind, status, accountID, model, page)
	if err != nil {
		response.InternalError(c, "读取采集日志失败")
		return
	}
	response.Success(c, map[string]any{"run": run, "items": items, "total": total, "page": page})
}
