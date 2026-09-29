package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient/tlsfingerprint"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	routingprovider "github.com/TokenFlux/TokenRouter/internal/routing/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type availableModelsAdminService struct {
	*managementMutationFixture
	provider providercore.Record
}

func (s *availableModelsAdminService) GetProvider(_ context.Context, id int64) (*providercore.Record, error) {
	if s.provider.ID == id {
		acc := s.provider
		return &acc, nil
	}
	return s.managementMutationFixture.GetProvider(context.Background(), id)
}

func setupAvailableModelsRouter(adminSvc ProviderManagement) *gin.Engine {
	router := gin.New()
	handler := NewManagementHandler(adminSvc, ManagementOptions{Catalog: routing.NewAdminCatalog(routingprovider.AdminCatalogOptions()), ModelDefaults: provideradapter.ModelDefaults()})
	router.GET("/api/v1/admin/providers/:id/models", handler.GetAvailableModels)
	return router
}

func TestAccountHandlerModelsNewProvidersPreserveOwnCatalogue(t *testing.T) {
	for platform, expected := range map[string]string{providercore.PlatformMiniMax: "MiniMax-M3", providercore.PlatformOpenCodeGo: "glm-5.3"} {
		for _, explicit := range []bool{false, true} {
			account := providercore.Record{ID: 1, Platform: platform, Type: providercore.ProviderTypeAPIKey}
			if explicit {
				account.Credentials = map[string]any{"model_whitelist": []any{"custom-model"}}
			}
			router := setupAvailableModelsRouter(&availableModelsAdminService{provider: account})
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/admin/providers/1/models", nil))
			require.Equal(t, http.StatusOK, w.Code)
			if explicit {
				require.Contains(t, w.Body.String(), "custom-model")
				require.NotContains(t, w.Body.String(), expected)
			} else {
				require.Contains(t, w.Body.String(), expected)
			}
		}
	}
}

type syncUpstreamHTTPUpstream struct {
	resp *http.Response
	err  error
}

func (u *syncUpstreamHTTPUpstream) Do(req *http.Request, proxyURL string, providerID int64, providerConcurrency int) (*http.Response, error) {
	if u.err != nil {
		return nil, u.err
	}
	return u.resp, nil
}

func (u *syncUpstreamHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, providerID int64, providerConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, providerID, providerConcurrency)
}

func setupSyncUpstreamModelsRouter(adminSvc ProviderManagement, upstream provideradapter.QoderTransport) *gin.Engine {
	router := gin.New()
	modelPolicy := egress.OperatorURLPolicy{}
	catalogue := &provideradapter.ModelCatalogue{Transport: upstream, Options: provideradapter.ModelCatalogueOptions{ValidateURL: modelPolicy.Validate, OperatorValidator: modelPolicy.Validate, BodyLimit: 8 * 1024 * 1024, CodexModelsURL: provideradapter.DefaultCodexModelsURL}}
	models := providercore.NewModelSyncService(catalogue.FetchUpstreamSupportedModels)
	handler := NewManagementHandler(adminSvc, ManagementOptions{Models: models})
	router.POST("/api/v1/admin/providers/:id/models/sync-upstream", handler.SyncUpstreamModels)
	router.POST("/api/v1/admin/providers/models/sync-upstream-preview", handler.SyncUpstreamModelsPreview)
	return router
}

func TestProviderHandlerGetAvailableModels_OpenAIOAuthUsesExplicitModelWhitelist(t *testing.T) {
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:       42,
			Name:     "openai-oauth",
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Status:   billing.StatusActive,
			Credentials: map[string]any{
				"model_whitelist": []any{"gpt-5"},
			},
		},
	}
	router := setupAvailableModelsRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers/42/models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	require.Equal(t, "gpt-5", resp.Data[0].ID)
}

func TestProviderHandlerGetAvailableModels_OpenAIOAuthMergesMappingAndWhitelistModels(t *testing.T) {
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:       44,
			Name:     "openai-oauth-merged-model-scope",
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Status:   billing.StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gpt-4.1": "gpt-5",
				},
				"model_whitelist": []any{"gpt-5", "gpt-5-mini"},
			},
		},
	}
	router := setupAvailableModelsRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers/44/models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	ids := make([]string, 0, len(resp.Data))
	for _, model := range resp.Data {
		ids = append(ids, model.ID)
	}
	require.ElementsMatch(t, []string{"gpt-4.1", "gpt-5", "gpt-5-mini"}, ids)
}

func TestProviderHandlerGetAvailableModels_OpenAIOAuthMappingOnlyFallsBackToDefaults(t *testing.T) {
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:       45,
			Name:     "openai-oauth-mapping-only",
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Status:   billing.StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gpt-4.1": "gpt-5",
				},
			},
		},
	}
	router := setupAvailableModelsRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers/45/models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data)
	require.Greater(t, len(resp.Data), 1)
}

func TestProviderHandlerGetAvailableModels_OpenAIOAuthExplicitEmptyWhitelistSkipsLegacySelfMappingFallback(t *testing.T) {
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:       46,
			Name:     "openai-oauth-explicit-empty-whitelist",
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Status:   billing.StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gpt-5": "gpt-5",
				},
				"model_whitelist": []any{},
			},
		},
	}
	router := setupAvailableModelsRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers/46/models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data)
	require.Greater(t, len(resp.Data), 1)
}

func TestProviderHandlerGetAvailableModels_OpenAIOAuthPassthroughFallsBackToDefaults(t *testing.T) {
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:       43,
			Name:     "openai-oauth-passthrough",
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Status:   billing.StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gpt-5": "gpt-5.1",
				},
			},
			Extra: map[string]any{
				"openai_passthrough": true,
			},
		},
	}
	router := setupAvailableModelsRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers/43/models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data)
	require.NotEqual(t, "gpt-5", resp.Data[0].ID)
}

func TestProviderHandlerGetAvailableModels_OpenAIAPIKeyDefaultsToConcreteGPT56Sol(t *testing.T) {
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:       46,
			Name:     "openai-apikey",
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Status:   billing.StatusActive,
			Credentials: map[string]any{
				"api_key": "test-key",
			},
		},
	}
	router := setupAvailableModelsRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers/46/models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data)
	ids := make([]string, 0, len(resp.Data))
	for _, model := range resp.Data {
		ids = append(ids, model.ID)
	}
	require.Contains(t, ids, "gpt-5.6-sol")
	require.NotContains(t, ids, "gpt-5.6")
}

func TestProviderHandlerGetAvailableModels_OpenAISparkShadowReturnsMappingModels(t *testing.T) {
	parentID := int64(100)
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:               44,
			Name:             "openai-spark-shadow",
			Platform:         capability.PlatformOpenAI,
			Type:             capability.ProviderTypeOAuth,
			Status:           billing.StatusActive,
			ParentProviderID: &parentID,
			QuotaDimension:   providercore.QuotaDimensionSpark,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gpt-5.3-codex-spark": "gpt-5.3-codex-spark",
				},
			},
		},
	}
	router := setupAvailableModelsRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers/44/models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	ids := make([]string, 0, len(resp.Data))
	for _, m := range resp.Data {
		ids = append(ids, m.ID)
	}
	require.ElementsMatch(t, []string{
		"gpt-5.3-codex-spark",
	}, ids, "影子可用模型由 model_mapping 派生（非写死）")
}

// TestProviderHandlerGetAvailableModels_GeminiGoogleOneUsesConservativeCatalog 验证
// Google One 旧 OAuth 通道只展示其仍支持的保守模型目录。
func TestProviderHandlerGetAvailableModels_GeminiGoogleOneUsesConservativeCatalog(t *testing.T) {
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:       45,
			Name:     "google-one",
			Platform: capability.PlatformGemini,
			Type:     capability.ProviderTypeOAuth,
			Status:   billing.StatusActive,
			Credentials: map[string]any{
				"oauth_type": "google_one",
			},
		},
	}
	router := setupAvailableModelsRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers/45/models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	ids := make([]string, 0, len(resp.Data))
	for _, model := range resp.Data {
		ids = append(ids, model.ID)
	}
	require.ElementsMatch(t, []string{"gemini-2.0-flash", "gemini-2.5-flash", "gemini-2.5-pro"}, ids)
	require.NotContains(t, ids, "gemini-3.5-flash")
	require.NotContains(t, ids, "gemini-2.5-flash-image")
}

func TestProviderHandlerGetAvailableModels_QoderFallsBackToDefaults(t *testing.T) {
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:       47,
			Name:     "qoder-cosy",
			Platform: capability.PlatformQoder,
			Type:     capability.ProviderTypeCosy,
			Status:   billing.StatusActive,
		},
	}
	router := setupAvailableModelsRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers/47/models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	ids := make([]string, 0, len(resp.Data))
	for _, model := range resp.Data {
		ids = append(ids, model.ID)
	}
	require.ElementsMatch(t, qoder.DefaultRequestModelIDsForSite(qoder.SiteGlobal), ids)
	require.NotContains(t, ids, "ultimate")
	require.NotContains(t, ids, "qmodel_latest")
	require.NotContains(t, ids, "quest-ultimate")
}

func TestProviderHandlerGetAvailableModels_QoderCNUsesCNSiteDefaults(t *testing.T) {
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:          49,
			Name:        "qoder-cn",
			Platform:    capability.PlatformQoder,
			Type:        capability.ProviderTypeCosy,
			Status:      billing.StatusActive,
			Credentials: map[string]any{"site": "cn"},
		},
	}
	router := setupAvailableModelsRouter(svc)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers/49/models", nil)
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var responseBody struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	ids := make([]string, 0, len(responseBody.Data))
	for _, model := range responseBody.Data {
		ids = append(ids, model.ID)
	}
	require.ElementsMatch(t, qoder.DefaultRequestModelIDsForSite(qoder.SiteCN), ids)
	require.NotContains(t, ids, "claude-opus-4-6")
	require.Contains(t, ids, "qwen3.6-flash")
}

func TestProviderHandlerGetAvailableModels_QoderUsesConfiguredModels(t *testing.T) {
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:       48,
			Name:     "qoder-cosy-custom",
			Platform: capability.PlatformQoder,
			Type:     capability.ProviderTypeCosy,
			Status:   billing.StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"custom-qoder-model": "qmodel",
				},
				"model_whitelist": []any{"qmodel"},
			},
		},
	}
	router := setupAvailableModelsRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/providers/48/models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	ids := make([]string, 0, len(resp.Data))
	for _, model := range resp.Data {
		ids = append(ids, model.ID)
	}
	require.ElementsMatch(t, []string{"custom-qoder-model", "qmodel"}, ids,
		"显式白名单目标和可请求别名共同出现在模型目录")
}

func TestProviderHandlerSyncUpstreamModels_ConfigErrorReturnsBadRequest(t *testing.T) {
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:       44,
			Name:     "openai-apikey-missing-key",
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Status:   billing.StatusActive,
			Credentials: map[string]any{
				"base_url": "https://openai.example.com/v1",
			},
		},
	}
	router := setupSyncUpstreamModelsRouter(svc, &syncUpstreamHTTPUpstream{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/44/models/sync-upstream", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "No OpenAI API key is available")
}

func TestProviderHandlerSyncUpstreamModels_UpstreamErrorDoesNotExposeBody(t *testing.T) {
	svc := &availableModelsAdminService{
		managementMutationFixture: newManagementMutationFixture(),
		provider: providercore.Record{
			ID:       45,
			Name:     "openai-apikey-upstream-error",
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Status:   billing.StatusActive,
			Credentials: map[string]any{
				"api_key":  "openai-key",
				"base_url": "https://openai.example.com/v1",
			},
		},
	}
	upstream := &syncUpstreamHTTPUpstream{resp: &http.Response{
		StatusCode: http.StatusBadGateway,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":"SECRET_TOKEN should not be exposed"}`)),
	}}
	router := setupSyncUpstreamModelsRouter(svc, upstream)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/45/models/sync-upstream", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Contains(t, rec.Body.String(), "Upstream model list request failed with HTTP 502")
	require.NotContains(t, rec.Body.String(), "SECRET_TOKEN")
}

func TestProviderHandlerSyncUpstreamModelsPreview_UsesProvidedCredentials(t *testing.T) {
	router := setupSyncUpstreamModelsRouter(newManagementMutationFixture(), &syncUpstreamHTTPUpstream{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"gpt-5.1"},{"id":"gpt-5.1"},{"id":"o3"}]}`)),
	}})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/providers/models/sync-upstream-preview",
		strings.NewReader(`{"platform":"openai","type":"apikey","base_url":"https://openai.example.com/v1","api_key":"openai-key"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Models []string `json:"models"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, []string{"gpt-5.1", "o3"}, resp.Data.Models)
}

func TestProviderHandlerSyncUpstreamModelsPreview_ConfigErrorReturnsBadRequest(t *testing.T) {
	router := setupSyncUpstreamModelsRouter(newManagementMutationFixture(), &syncUpstreamHTTPUpstream{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/providers/models/sync-upstream-preview",
		strings.NewReader(`{"platform":"openai","type":"apikey","base_url":"https://openai.example.com/v1"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "required")
}
