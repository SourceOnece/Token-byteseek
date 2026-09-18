package admin

import (
	"github.com/TokenFlux/TokenRouter/internal/pkg/response"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

// 管理员当前页批量查询，沿用 GET 的只读语义；不把查询变成手动探测入口。
func (h *SettingHandler) GetCodexTicketStatus(c *gin.Context) {
	if h.codexTickets == nil {
		response.InternalError(c, "票据服务不可用")
		return
	}
	raw := strings.Split(c.Query("account_ids"), ",")
	if len(raw) == 0 || len(raw) > 100 {
		response.BadRequest(c, "每次最多查询 100 个账号")
		return
	}
	ids := make([]int64, 0, len(raw))
	for _, item := range raw {
		id, err := strconv.ParseInt(strings.TrimSpace(item), 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "无效的账号 ID")
			return
		}
		ids = append(ids, id)
	}
	status, err := h.codexTickets.Status(c.Request.Context(), ids)
	if err != nil {
		response.InternalError(c, "读取票据状态失败")
		return
	}
	response.Success(c, status)
}

// 独立管理员设置入口，不扩散到公开设置或整页设置响应，代理只写不回显。
func (h *SettingHandler) SetCodexTicketService(s *service.CodexTicketService) { h.codexTickets = s }
func (h *SettingHandler) GetCodexTicketSettings(c *gin.Context) {
	if h.codexTickets == nil {
		response.InternalError(c, "票据服务不可用")
		return
	}
	settings, err := h.codexTickets.View(c.Request.Context())
	if err != nil {
		response.InternalError(c, "读取票据设置失败")
		return
	}
	response.Success(c, settings)
}
func (h *SettingHandler) UpdateCodexTicketSettings(c *gin.Context) {
	if h.codexTickets == nil {
		response.InternalError(c, "票据服务不可用")
		return
	}
	var input service.CodexTicketSettingsUpdate
	if c.ShouldBindJSON(&input) != nil {
		response.BadRequest(c, "无效的票据设置")
		return
	}
	settings, err := h.codexTickets.Update(c.Request.Context(), input)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, settings)
}
