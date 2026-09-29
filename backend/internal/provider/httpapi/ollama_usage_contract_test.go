package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type ollamaCloudUsageHandlerTestRepo struct {
	provider          *providercore.Record
	providers         []*providercore.Record
	groupResolveCalls int
}

func (r *ollamaCloudUsageHandlerTestRepo) GetByID(_ context.Context, id int64) (*providercore.Record, error) {
	if r.provider != nil && r.provider.ID == id {
		return r.provider, nil
	}
	for _, provider := range r.providers {
		if provider.ID == id {
			return provider, nil
		}
	}
	return nil, providercore.ErrProviderNotFound
}

func (r *ollamaCloudUsageHandlerTestRepo) ListOllamaCloudUsageGroupProviders(_ context.Context, _ []*providercore.Record) ([]providercore.Record, error) {
	r.groupResolveCalls++
	result := make([]providercore.Record, 0, len(r.providers)+1)
	if r.provider != nil {
		result = append(result, *r.provider)
	}
	for _, provider := range r.providers {
		result = append(result, *provider)
	}
	return result, nil
}

func (r *ollamaCloudUsageHandlerTestRepo) SaveOllamaCloudUsageSession(context.Context, *providercore.Record, string, bool) error {
	return nil
}

func (r *ollamaCloudUsageHandlerTestRepo) DeleteOllamaCloudUsageSession(context.Context, *providercore.Record) error {
	return nil
}

func (r *ollamaCloudUsageHandlerTestRepo) SetOllamaCloudUsageAutoRefresh(context.Context, *providercore.Record, bool) error {
	return nil
}

func (r *ollamaCloudUsageHandlerTestRepo) UpdateOllamaCloudUsageSnapshot(context.Context, *providercore.Record, *providercore.OllamaCloudUsageSnapshot) error {
	return nil
}

func (r *ollamaCloudUsageHandlerTestRepo) DisableOllamaCloudUsageAutoRefresh(context.Context, *providercore.Record) error {
	return nil
}

func (r *ollamaCloudUsageHandlerTestRepo) ListDueOllamaCloudUsageProviders(context.Context, time.Time, time.Duration, time.Duration, int) ([]providercore.Record, error) {
	return nil, nil
}

func newOllamaCloudUsageHandlerTestService(t *testing.T) *providercore.OllamaCloudUsageService {
	t.Helper()
	svc := providercore.NewOllamaCloudUsageService(nil, nil, nil, providercore.OllamaUsageOptions{})
	t.Cleanup(svc.Stop)
	return svc
}

func newOllamaCloudUsageHandlerContext(method, target, body, id string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	if id != "" {
		ctx.Params = gin.Params{{Key: "id", Value: id}}
	}
	return ctx, recorder
}

func TestOllamaCloudUsageHandlersValidateRequestsAndDependencies(t *testing.T) {
	svc := newOllamaCloudUsageHandlerTestService(t)

	t.Run("invalid provider id", func(t *testing.T) {
		ctx, recorder := newOllamaCloudUsageHandlerContext(http.MethodGet, "/admin/providers/not-an-id/ollama-cloud-usage", "", "not-an-id")
		NewOllamaUsageHandler(svc).GetOllamaCloudUsage(ctx)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("empty session", func(t *testing.T) {
		ctx, recorder := newOllamaCloudUsageHandlerContext(http.MethodPut, "/admin/providers/7/ollama-cloud-usage/session", `{"session":""}`, "7")
		NewOllamaUsageHandler(svc).SaveOllamaCloudUsageSession(ctx)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("missing enabled", func(t *testing.T) {
		ctx, recorder := newOllamaCloudUsageHandlerContext(http.MethodPut, "/admin/providers/7/ollama-cloud-usage/auto-refresh", `{}`, "7")
		NewOllamaUsageHandler(svc).SetOllamaCloudUsageAutoRefresh(ctx)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("service unavailable", func(t *testing.T) {
		ctx, recorder := newOllamaCloudUsageHandlerContext(http.MethodGet, "/admin/providers/7/ollama-cloud-usage", "", "7")
		NewOllamaUsageHandler(nil).GetOllamaCloudUsage(ctx)
		require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
		require.Contains(t, recorder.Body.String(), "OLLAMA_CLOUD_USAGE_UNAVAILABLE")
	})
}

func TestOllamaCloudUsageEncryptionKeyStateConsistentAcrossProviderResponses(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run("configured="+strconv.FormatBool(configured), func(t *testing.T) {
			provider := &providercore.Record{
				ID:          7,
				Name:        "ollama",
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeAPIKey,
				Credentials: map[string]any{"base_url": "https://ollama.com", "api_key": "test-key"},
				Extra:       map[string]any{},
				Status:      providercore.StatusActive,
			}
			adminService := &ollamaManagementFixture{}
			adminService.providers = []providercore.Record{*provider}
			usageService := providercore.NewOllamaCloudUsageService(
				&ollamaCloudUsageHandlerTestRepo{provider: provider}, nil, nil, providercore.OllamaUsageOptions{EncryptionKeyConfigured: configured},
			)
			t.Cleanup(usageService.Stop)

			handler := newOllamaManagementHandler(adminService, usageService)
			router := gin.New()
			router.GET("/providers", handler.List)
			router.GET("/providers/:id", handler.GetByID)
			router.GET("/providers/:id/ollama-cloud-usage", NewOllamaUsageHandler(usageService).GetOllamaCloudUsage)

			listRecorder := httptest.NewRecorder()
			router.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/providers?page=1&page_size=20", nil))
			require.Equal(t, http.StatusOK, listRecorder.Code)
			var listPayload struct {
				Data struct {
					Items []struct {
						OllamaCloudUsage *providercore.OllamaCloudUsageState `json:"ollama_cloud_usage"`
					} `json:"items"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(listRecorder.Body.Bytes(), &listPayload))
			require.Len(t, listPayload.Data.Items, 1)
			require.NotNil(t, listPayload.Data.Items[0].OllamaCloudUsage)

			detailRecorder := httptest.NewRecorder()
			router.ServeHTTP(detailRecorder, httptest.NewRequest(http.MethodGet, "/providers/7", nil))
			require.Equal(t, http.StatusOK, detailRecorder.Code)
			var detailPayload struct {
				Data struct {
					OllamaCloudUsage *providercore.OllamaCloudUsageState `json:"ollama_cloud_usage"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(detailRecorder.Body.Bytes(), &detailPayload))
			require.NotNil(t, detailPayload.Data.OllamaCloudUsage)

			stateRecorder := httptest.NewRecorder()
			router.ServeHTTP(stateRecorder, httptest.NewRequest(http.MethodGet, "/providers/7/ollama-cloud-usage", nil))
			require.Equal(t, http.StatusOK, stateRecorder.Code)
			var statePayload struct {
				Data providercore.OllamaCloudUsageState `json:"data"`
			}
			require.NoError(t, json.Unmarshal(stateRecorder.Body.Bytes(), &statePayload))

			listConfigured := listPayload.Data.Items[0].OllamaCloudUsage.EncryptionKeyConfigured
			detailConfigured := detailPayload.Data.OllamaCloudUsage.EncryptionKeyConfigured
			require.Equal(t, configured, listConfigured)
			require.Equal(t, statePayload.Data.EncryptionKeyConfigured, listConfigured)
			require.Equal(t, statePayload.Data.EncryptionKeyConfigured, detailConfigured)
		})
	}
}

func TestOllamaCloudUsageSharedStateMatchesListDetailAndSpecialEndpointWithoutListNPlusOne(t *testing.T) {
	now := time.Now().UTC()
	source := &providercore.Record{
		ID: 7, Name: "source", Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://ollama.com", "api_key": "shared-secret-key"},
		Extra: map[string]any{
			providercore.OllamaCloudUsageSessionExtraKey:     "ciphertext-secret",
			providercore.OllamaCloudUsageAutoRefreshExtraKey: true,
			providercore.OllamaCloudUsageSnapshotExtraKey: &providercore.OllamaCloudUsageSnapshot{
				Status: providercore.OllamaCloudUsageStatusOK, Data: &providercore.OllamaCloudUsageData{Plan: "pro"},
				LastAttemptAt: now, NextRefreshAt: now.Add(time.Hour),
			},
		},
		Status: providercore.StatusActive,
	}
	sibling := &providercore.Record{
		ID: 8, Name: "sibling", Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeAPIKey,
		Credentials: map[string]any{"base_url": "HTTPS://WWW.OLLAMA.COM:443/v1", "api_key": "shared-secret-key"},
		Extra:       map[string]any{}, Status: providercore.StatusActive,
	}
	repo := &ollamaCloudUsageHandlerTestRepo{providers: []*providercore.Record{source, sibling}}
	adminService := &ollamaManagementFixture{}
	adminService.providers = []providercore.Record{*source, *sibling}
	usageService := providercore.NewOllamaCloudUsageService(repo, nil, nil, providercore.OllamaUsageOptions{EncryptionKeyConfigured: true})
	t.Cleanup(usageService.Stop)
	handler := newOllamaManagementHandler(adminService, usageService)
	router := gin.New()
	router.GET("/providers", handler.List)
	router.GET("/providers/:id", handler.GetByID)
	router.GET("/providers/:id/ollama-cloud-usage", NewOllamaUsageHandler(usageService).GetOllamaCloudUsage)

	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/providers?page=1&page_size=20", nil))
	require.Equal(t, http.StatusOK, listRecorder.Code)
	require.Equal(t, 1, repo.groupResolveCalls, "the full list page must use one group-resolution batch")
	var listPayload struct {
		Data struct {
			Items []struct {
				ID               int64                               `json:"id"`
				OllamaCloudUsage *providercore.OllamaCloudUsageState `json:"ollama_cloud_usage"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRecorder.Body.Bytes(), &listPayload))
	require.Len(t, listPayload.Data.Items, 2)
	for _, item := range listPayload.Data.Items {
		require.True(t, item.OllamaCloudUsage.Configured)
		require.Equal(t, "pro", item.OllamaCloudUsage.Snapshot.Data.Plan)
	}

	detailRecorder := httptest.NewRecorder()
	router.ServeHTTP(detailRecorder, httptest.NewRequest(http.MethodGet, "/providers/8", nil))
	require.Equal(t, http.StatusOK, detailRecorder.Code)
	var detailPayload struct {
		Data struct {
			OllamaCloudUsage *providercore.OllamaCloudUsageState `json:"ollama_cloud_usage"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(detailRecorder.Body.Bytes(), &detailPayload))

	stateRecorder := httptest.NewRecorder()
	router.ServeHTTP(stateRecorder, httptest.NewRequest(http.MethodGet, "/providers/8/ollama-cloud-usage", nil))
	require.Equal(t, http.StatusOK, stateRecorder.Code)
	var statePayload struct {
		Data providercore.OllamaCloudUsageState `json:"data"`
	}
	require.NoError(t, json.Unmarshal(stateRecorder.Body.Bytes(), &statePayload))
	require.Equal(t, statePayload.Data.Configured, detailPayload.Data.OllamaCloudUsage.Configured)
	require.Equal(t, statePayload.Data.Snapshot, detailPayload.Data.OllamaCloudUsage.Snapshot)
	for _, body := range []string{listRecorder.Body.String(), detailRecorder.Body.String(), stateRecorder.Body.String()} {
		require.NotContains(t, body, "shared-secret-key")
		require.NotContains(t, body, "ciphertext-secret")
	}
}

func TestGetOllamaCloudUsageSettingsHandlerSuccess(t *testing.T) {
	ctx, recorder := newOllamaCloudUsageHandlerContext(http.MethodGet, "/admin/providers/ollama-cloud-usage/settings", "", "")
	handler := NewOllamaUsageHandler(newOllamaCloudUsageHandlerTestService(t))

	handler.GetOllamaCloudUsageSettings(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"enabled":false`)
	require.Contains(t, recorder.Body.String(), `"interval_minutes":60`)
	require.Contains(t, recorder.Body.String(), `"debounce_minutes":1`)
}
