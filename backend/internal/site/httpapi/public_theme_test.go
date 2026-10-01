package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/pkg/timezone"
	"github.com/TokenFlux/TokenRouter/internal/site"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type themePublicSource struct{ value string }

func (s themePublicSource) LoadSitePublicInputs(context.Context) (site.PublicInputs, error) {
	return site.PublicInputs{Values: map[string]string{site.SettingKeySiteTheme: s.value}}, nil
}
func (themePublicSource) PublicVersion() string { return "theme-test" }

// API 与 HTML 注入必须一致，老部署无配置和历史脏值都使用同一默认值。
func TestSiteThemeMatchesPublicAPIAndInjection(t *testing.T) {
	for _, test := range []struct{ value, want string }{{"", "tokenflux"}, {"tokenflux", "tokenflux"}, {"bauhaus", "bauhaus"}, {"invalid", "tokenflux"}} {
		t.Run(test.value, func(t *testing.T) {
			service := site.NewPublicService(themePublicSource{test.value}, timezone.NewCalendar(time.UTC), "UTC")
			response := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(response)
			ctx.Request = httptest.NewRequest("GET", "/settings/public", nil)
			NewPublicHandler(service, "theme-test").GetPublicSettings(ctx)
			require.Equal(t, 200, response.Code)
			var body struct {
				Data map[string]any `json:"data"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			require.Equal(t, test.want, body.Data["site_theme"])
			injection, err := service.GetPublicSettingsForInjection(context.Background())
			require.NoError(t, err)
			encoded, err := json.Marshal(injection)
			require.NoError(t, err)
			var fields map[string]any
			require.NoError(t, json.Unmarshal(encoded, &fields))
			require.Equal(t, test.want, fields["site_theme"])
		})
	}
	require.Contains(t, site.PublicInputKeys(), site.SettingKeySiteTheme)
	require.Contains(t, site.PublicValueKeys(), site.SettingKeySiteTheme)
}
