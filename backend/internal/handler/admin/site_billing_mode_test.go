//go:build unit

package admin

import (
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

// 部分配置保存不能重置管理员已经选择的模式。
func TestSiteBillingModePartialUpdate(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{service.SettingKeySubscriptionEnabled: "true"})
	rec := doUpdateSettings(t, h, map[string]any{"subscription_enabled": false}, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "false", repo.values[service.SettingKeySubscriptionEnabled])
	rec = doUpdateSettings(t, h, map[string]any{"site_name": "Example"}, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "false", repo.values[service.SettingKeySubscriptionEnabled])
}
