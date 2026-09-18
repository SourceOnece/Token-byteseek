//go:build unit

package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 未确认/不合法请求在进入采集服务前拒绝，不从 GET 触发有副作用操作。
func TestCodexTicketManualHandlerRejectsUnconfirmed(t *testing.T) {
	for _, body := range []string{`{"account_ids":[1]}`, `{"account_ids":[],"confirmed":true}`, `{"account_ids":[-1],"confirmed":true}`, `invalid`} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		h := &SettingHandler{codexTickets: &service.CodexTicketService{}}
		h.BatchCodexTicketCollect(c)
		require.Equal(t, 400, w.Code)
	}
}

func TestCodexTicketHistoryDeleteRejectsAmbiguousTargets(t *testing.T) {
	for _, params := range []gin.Params{
		{{Key: "id", Value: "invalid"}},
		{{Key: "id", Value: "11111111-1111-4111-8111-111111111111"}, {Key: "event_id", Value: "0"}},
		{{Key: "id", Value: "11111111-1111-4111-8111-111111111111"}, {Key: "event_id", Value: "-1"}},
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
		c.Params = params
		(&SettingHandler{}).DeleteCodexTicketHistory(c)
		require.Equal(t, 400, w.Code)
	}
}
