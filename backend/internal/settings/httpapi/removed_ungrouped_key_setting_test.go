package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 不访问设置存储就拒绝已移除的旁路开关。
func TestUpdateSettingsRejectsUngroupedScheduling(t *testing.T) {
	for _, value := range []string{"true", "false", "null"} {
		router := gin.New()
		router.PUT("/settings", (&Handler{}).UpdateSettings)
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(`{"allow_ungrouped_key_scheduling":`+value+`}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Contains(t, recorder.Body.String(), "REMOVED_SETTING_FIELD")
	}
}
