//go:build unit

package app

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/site"
	"github.com/stretchr/testify/require"
)

// 复用真实综合设置装配，覆盖站点键持久化、响应、旧客户端省略和非法写入。
func TestSiteThemeSettingsRoundTrip(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{site.SettingKeySiteTheme: site.SiteThemeTokenFlux})
	rec := doUpdateSettings(t, h, map[string]any{"site_theme": "bauhaus"}, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "bauhaus", repo.values[site.SettingKeySiteTheme])
	var result struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Equal(t, "bauhaus", result.Data["site_theme"])

	rec = doUpdateSettings(t, h, map[string]any{"site_name": "updated"}, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "bauhaus", repo.values[site.SettingKeySiteTheme], "省略不能重置全站皮肤")
	for _, value := range []any{"custom-css", "<script>", 42} {
		rec = doUpdateSettings(t, h, map[string]any{"site_theme": value, "site_name": "must-not-save"}, nil)
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		require.Equal(t, "bauhaus", repo.values[site.SettingKeySiteTheme])
		require.Equal(t, "updated", repo.values[site.SettingKeySiteName])
	}
	rec = doUpdateSettings(t, h, map[string]any{"site_theme": "tokenflux"}, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "tokenflux", repo.values[site.SettingKeySiteTheme])
}
