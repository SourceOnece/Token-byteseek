package httpapi

import (
	"context"
	"strconv"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	response "github.com/TokenFlux/TokenRouter/internal/server/httpx"
	"github.com/gin-gonic/gin"
)

// TestHandler 绑定管理 HTTP 字段、SSE 输出和连接测试成功后的恢复端口。
type TestHandler struct {
	tests   *provider.TestService
	recover func(context.Context, int64) error
}

func NewTestHandler(tests *provider.TestService, recover func(context.Context, int64) error) *TestHandler {
	return &TestHandler{tests: tests, recover: recover}
}

// TestProviderRequest 表示提供商连接测试的请求体。
type TestProviderRequest struct {
	ModelID string `json:"model_id"`
	Prompt  string `json:"prompt"`
	Mode    string `json:"mode"`
	// Protocol 只作用于本次文字测试：OpenAI 选择 Responses 或 Chat，国产平台选择已启用的原生协议。
	Protocol string `json:"protocol"`
	// TestType 由管理端明确指定测试文字或图片，避免服务端猜测模型能力。
	TestType string `json:"test_type"`
	// TestMode 兼容早期客户端使用的字段名，优先级低于 test_type。
	TestMode string `json:"test_mode"`
}

// Test handles testing provider connectivity with SSE streaming
// POST /api/v1/admin/providers/:id/test
func (h *TestHandler) Test(c *gin.Context) {
	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid provider ID")
		return
	}

	var req TestProviderRequest
	// Allow empty body, model_id is optional
	_ = c.ShouldBindJSON(&req)

	// 使用唯一原生测试用例，HTTP 输出器同步写入 SSE 事件。
	testType := req.TestType
	if testType == "" {
		testType = req.TestMode
	}
	if err := h.tests.Test(c.Request.Context(), provider.TestRequest{ProviderID: providerID, Model: req.ModelID, Prompt: req.Prompt, Mode: req.Mode, Type: &testType, Protocol: req.Protocol, UserAgent: c.GetHeader("User-Agent"), Originator: c.GetHeader("originator")}, NewTestEventSink(c.Writer)); err != nil {
		// Error already sent via SSE, just log
		return
	}

	if h.recover != nil {
		if err := h.recover(c.Request.Context(), providerID); err != nil {
			_ = c.Error(err)
		}
	}
}
