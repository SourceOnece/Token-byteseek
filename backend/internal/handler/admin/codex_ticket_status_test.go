package admin

import (
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 输入在触碰服务前校验，拒绝空值/负数/溢出和无界列表，不依赖生产账号。
func TestCodexTicketStatusRejectsInvalidIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &SettingHandler{codexTickets: &service.CodexTicketService{}}
	for _, ids := range []string{"", "0", "-1", "oops", "1,", "9223372036854775808", strings.Repeat("1,", 100) + "1"} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/codex-ticket/status?account_ids="+ids, nil)
		h.GetCodexTicketStatus(c)
		require.Equal(t, http.StatusBadRequest, w.Code)
	}
}
