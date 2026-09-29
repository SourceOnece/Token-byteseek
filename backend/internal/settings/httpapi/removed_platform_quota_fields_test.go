package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 旧字段在读取其它设置和执行保存前拒绝，包括显式 null。
func TestUpdateSettingsRejectsRemovedPlatformQuotas(t *testing.T) {
	for _, name := range []string{"default_platform_quotas", "auth_source_default_email_platform_quotas", "auth_source_default_linuxdo_platform_quotas", "auth_source_default_oidc_platform_quotas", "auth_source_default_wechat_platform_quotas", "auth_source_default_github_platform_quotas", "auth_source_default_google_platform_quotas", "auth_source_default_dingtalk_platform_quotas"} {
		t.Run(name, func(t *testing.T) {
			router := gin.New()
			router.PUT("/settings", (&Handler{}).UpdateSettings)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(`{"`+name+`":null}`))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), "REMOVED_SETTING_FIELD")
			require.Contains(t, recorder.Body.String(), name)
		})
	}
}
