package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 报告设置中的旧嵌套字段必须拒绝，避免管理员误以为保存成功。
func TestEmailReportRejectsLegacyProviderFields(t *testing.T) {
	handler := &OpsHandler{opsService: newRuntimeOpsService(t)}
	router := gin.New()
	router.PUT("/config", handler.UpdateEmailNotificationConfig)
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"report":{"account_health_enabled":false}}`, http.StatusBadRequest},
		{`{"report":{"provider_health_enabled":false}}`, http.StatusOK},
	} {
		request := httptest.NewRequest(http.MethodPut, "/config", strings.NewReader(tc.body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
	}
}
