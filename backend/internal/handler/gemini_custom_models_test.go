//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/pkg/gemini"
	"github.com/TokenFlux/TokenRouter/internal/server/middleware"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGeminiV1BetaListModels_CustomGroupListUsesNativeResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1beta/models", nil)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			Platform: service.PlatformGemini,
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"gemini-2.5-pro", "models/gemini-custom"},
			},
		},
	})

	(&GatewayHandler{}).GeminiV1BetaListModels(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var got gemini.ModelsListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, []gemini.Model{
		gemini.FallbackModel("gemini-2.5-pro"),
		gemini.FallbackModel("models/gemini-custom"),
	}, got.Models)
}

// 自定义分组列表沿用 Key 别名投影，保留目标元数据且不暴露列表外目标或通配符。
func TestGeminiV1BetaListModels_CustomGroupListPreservesAPIKeyAliases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1beta/models", nil)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		ModelMapping: map[string]string{
			"my-gemini":      "gemini-2.5-pro",
			"custom-alias":   "gemini-custom",
			"unlisted-alias": "gemini-2.5-flash",
			"wildcard-*":     "gemini-2.5-pro",
		},
		Group: &service.Group{
			Platform: service.PlatformGemini,
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"gemini-2.5-pro", "models/gemini-custom"},
			},
		},
	})

	(&GatewayHandler{}).GeminiV1BetaListModels(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var got gemini.ModelsListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Models, 4)
	pro := gemini.FallbackModel("gemini-2.5-pro")
	custom := gemini.FallbackModel("models/gemini-custom")
	require.Equal(t, []gemini.Model{pro, custom}, got.Models[:2])
	pro.Name, pro.DisplayName = "models/my-gemini", "my-gemini"
	custom.Name, custom.DisplayName = "models/custom-alias", "custom-alias"
	require.ElementsMatch(t, []gemini.Model{pro, custom}, got.Models[2:])
}

func TestGeminiV1BetaListModels_ForcedAntigravityIgnoresCustomGroupList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/antigravity/v1beta/models", nil)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			Platform: service.PlatformGemini,
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"gemini-custom"},
			},
		},
	})
	c.Set(string(middleware.ContextKeyForcePlatform), service.PlatformAntigravity)
	id := int64(45)
	key, _ := middleware.GetAPIKeyFromContext(c)
	key.GroupID = &id
	repo := &geminiAllowlistAccountRepoStub{gatewayModelsAccountRepoStub: gatewayModelsAccountRepoStub{byGroup: map[int64][]service.Account{id: {{ID: 1, Platform: service.PlatformAntigravity, Credentials: map[string]any{"model_mapping": map[string]any{"gemini-actual": "gemini-3.8-flash-high"}}}}}}}
	h := &GatewayHandler{geminiCompatService: service.NewGeminiMessagesCompatService(repo, nil, nil, nil, nil, nil, nil, nil, nil)}
	h.GeminiV1BetaListModels(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var got gemini.ModelsListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	// 本地账号映射保留默认模型并追加别名；专用路由不能被分组展示列表取代。
	require.Contains(t, got.Models, gemini.FallbackModel("gemini-actual"))
	require.NotContains(t, got.Models, gemini.FallbackModel("gemini-custom"))
}

func TestCustomGeminiModelsList_DisabledKeepsExistingFlow(t *testing.T) {
	group := &service.Group{
		ModelsListConfig: service.GroupModelsListConfig{
			Enabled: false,
			Models:  []string{"gemini-2.5-pro"},
		},
	}

	_, ok := customGeminiModelsList(group)
	require.False(t, ok)
}
