//go:build unit

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func ticketFailoverFixture() *service.UpstreamFailoverError {
	return &service.UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable,
		Stage: service.GatewayFailureStageAccountSelection, Scope: service.GatewayFailureScopeAccount,
		Reason: service.CodexTicketUnavailableReason, NextAccountAction: service.NextAccountRetry}
}

func TestCodexTicketFailoverExhaustionAllProtocols(t *testing.T) {
	for _, protocol := range []string{"openai", "messages", "responses_compat", "chat_compat"} {
		for _, streaming := range []bool{false, true} {
			t.Run(protocol+"/"+map[bool]string{false: "json", true: "stream"}[streaming], func(t *testing.T) {
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = httptest.NewRequest(http.MethodPost, "/responses", nil)
				if streaming {
					c.Writer.WriteHeaderNow()
				}
				err := ticketFailoverFixture()
				switch protocol {
				case "openai":
					(&OpenAIGatewayHandler{}).handleFailoverExhausted(c, err, streaming)
				case "messages":
					(&OpenAIGatewayHandler{}).handleAnthropicFailoverExhausted(c, err, streaming)
				case "responses_compat":
					(&GatewayHandler{}).handleResponsesFailoverExhausted(c, err, streaming)
				case "chat_compat":
					(&GatewayHandler{}).handleCCFailoverExhausted(c, err, streaming)
				}
				if !streaming {
					require.Equal(t, http.StatusServiceUnavailable, w.Code)
				}
				if protocol == "chat_compat" && streaming {
					// 通用Chat适配已开流时由转发层结束流，耗尽处理不得再拼接JSON。
					require.Empty(t, w.Body.String())
					return
				}
				require.Contains(t, w.Body.String(), "Service temporarily unavailable")
				require.NotContains(t, w.Body.String(), "Upstream request failed")
			})
		}
	}
}

func TestCodexTicketFailoverBudgetAndNoReplay(t *testing.T) {
	err := ticketFailoverFixture()
	state := NewFailoverState(1, false)
	state.HandleFailoverError(context.Background(), &mockTempUnscheduler{}, 1, service.PlatformOpenAI, 0, err)
	require.Contains(t, state.FailedAccountIDs, int64(1))
	require.Equal(t, FailoverExhausted, state.HandleFailoverError(context.Background(), &mockTempUnscheduler{}, 2, service.PlatformOpenAI, 0, err))
	require.False(t, shouldReportOpenAIWSProxyAccountFailure(service.ErrCodexTicketUnavailable))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	before := c.Writer.Size()
	require.True(t, openAIForwardMayFailover(c, before, err))
	_, _ = c.Writer.WriteString("data: an answer\n\n")
	require.False(t, openAIForwardMayFailover(c, before, err), "已输出回答不得跨号重放")
}

func TestCodexTicketSelectionExhaustedDoesNotRestartFailedAccounts(t *testing.T) {
	state := NewFailoverState(3, false)
	state.LastFailoverErr = ticketFailoverFixture()
	state.FailedAccountIDs[1] = struct{}{}
	// 不能套用上游503的单账号退避，否则清空排除列表后持续选择缺票账号。
	require.Equal(t, FailoverExhausted, state.HandleSelectionExhausted(context.Background()))
	require.Contains(t, state.FailedAccountIDs, int64(1))
}
