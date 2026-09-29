package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupProviderMutationContractRouter(adminSvc *managementMutationFixture) *gin.Engine {
	router := gin.New()
	providerHandler := newMutationHandler(adminSvc, nil)
	router.POST("/api/v1/admin/providers", providerHandler.Create)
	router.PUT("/api/v1/admin/providers/:id", providerHandler.Update)
	router.POST("/api/v1/admin/providers/bulk-update", providerHandler.BulkUpdate)
	return router
}

func TestProviderHandlerBatchRefreshSupportsQoderCosy(t *testing.T) {
	exchange := &providercore.ManualCredentialExchange{Qoder: func(_ context.Context, value *providercore.Record) (map[string]any, error) {
		require.Equal(t, "old-refresh", value.GetCredential("refresh_token"))
		require.Equal(t, "old-token", value.GetCredential("security_oauth_token"))
		require.Equal(t, "machine-1", value.GetCredential("machine_id"))
		return map[string]any{"security_oauth_token": "new-token", "refresh_token": "new-refresh", "machine_id": "machine-1"}, nil
	}}
	adminSvc := newManagementMutationFixture()
	adminSvc.providers = []providercore.Record{{
		ID:       44,
		Name:     "qoder",
		Platform: capability.PlatformQoder,
		Type:     capability.ProviderTypeCosy,
		Status:   billing.StatusActive,
		Credentials: map[string]any{
			"security_oauth_token": "old-token",
			"refresh_token":        "old-refresh",
			"machine_id":           "machine-1",
		},
	}}
	router := setupProviderMutationContractRouter(adminSvc)
	managed := newManagedRefreshFixture(adminSvc, exchange)
	handler := NewManagementHandler(adminSvc, ManagementOptions{Batch: providercore.NewManagementBatch(adminSvc, managed)})
	router.POST("/api/v1/admin/providers/batch-refresh", handler.BatchRefresh)

	body, _ := json.Marshal(map[string]any{"provider_ids": []int64{44}})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/batch-refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, float64(0), resp["code"])
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(1), data["success"])
	require.Equal(t, float64(0), data["failed"])
	require.Equal(t, "new-token", adminSvc.updateProviderInput.Credentials["security_oauth_token"])
	require.Equal(t, "new-refresh", adminSvc.updateProviderInput.Credentials["refresh_token"])
}

func TestBulkUpdateAcceptsFilterTargetRequest(t *testing.T) {
	adminSvc := newManagementMutationFixture()
	router := setupProviderMutationContractRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"filters": map[string]any{
			"platform":     "openai",
			"type":         "oauth",
			"status":       "active",
			"group":        "12",
			"privacy_mode": "blocked",
			"search":       "bulk-target",
		},
		"schedulable": true,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/bulk-update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, float64(0), resp["code"])
}

// 旧入口返回404，旧确认和默认组字段在任何写操作之前拒绝。
func TestProviderManagementRejectsRetiredGroupInputs(t *testing.T) {
	for _, endpoint := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/admin/providers", `{"name":"mixed","platform":"openai","type":"apikey","credentials":{"api_key":"test"}}`},
		{http.MethodPut, "/api/v1/admin/providers/3", `{"group_ids":[27]}`},
		{http.MethodPost, "/api/v1/admin/providers/bulk-update", `{"provider_ids":[1],"group_ids":[27]}`},
	} {
		for _, field := range []string{"confirm_mixed_channel_risk", "skip_default_group_bind"} {
			t.Run(endpoint.method+endpoint.path+field, func(t *testing.T) {
				source := newManagementMutationFixture()
				router := setupProviderMutationContractRouter(source)
				var payload map[string]any
				require.NoError(t, json.Unmarshal([]byte(endpoint.body), &payload))
				payload[field] = false
				body, err := json.Marshal(payload)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()
				req := httptest.NewRequest(endpoint.method, endpoint.path, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(recorder, req)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Contains(t, recorder.Body.String(), field)
			})
		}
	}
	router := setupProviderMutationContractRouter(newManagementMutationFixture())
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/check-mixed-channel", nil))
	require.Equal(t, http.StatusNotFound, recorder.Code)
}
