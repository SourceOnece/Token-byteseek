package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestProviderAdminBoundariesDiscardDeprecatedLongContextBillingExtra(t *testing.T) {
	const malformedExtra = `"extra":{"openai_long_context_billing_enabled":{"malformed":true},"preserved":"value"}`

	tests := []struct {
		name          string
		method        string
		path          string
		body          string
		mount         func(*gin.Engine, *ManagementHandler)
		capturedExtra func(*managementMutationFixture) map[string]any
	}{
		{
			name:   "create",
			method: http.MethodPost,
			path:   "/providers",
			body:   `{"name":"provider","platform":"openai","type":"apikey","credentials":{"api_key":"test"},` + malformedExtra + `}`,
			mount:  func(router *gin.Engine, handler *ManagementHandler) { router.POST("/providers", handler.Create) },
			capturedExtra: func(stub *managementMutationFixture) map[string]any {
				require.Len(t, stub.createdProviders, 1)
				return stub.createdProviders[0].Extra
			},
		},
		{
			name:   "update",
			method: http.MethodPut,
			path:   "/providers/1",
			body:   `{` + malformedExtra + `}`,
			mount:  func(router *gin.Engine, handler *ManagementHandler) { router.PUT("/providers/:id", handler.Update) },
			capturedExtra: func(stub *managementMutationFixture) map[string]any {
				require.NotNil(t, stub.updateProviderInput)
				return stub.updateProviderInput.Extra
			},
		},
		{
			name:   "bulk update",
			method: http.MethodPost,
			path:   "/providers/bulk-update",
			body:   `{"provider_ids":[1],` + malformedExtra + `}`,
			mount: func(router *gin.Engine, handler *ManagementHandler) {
				router.POST("/providers/bulk-update", handler.BulkUpdate)
			},
			capturedExtra: func(stub *managementMutationFixture) map[string]any {
				require.NotNil(t, stub.lastBulkUpdateProviderInput)
				return stub.lastBulkUpdateProviderInput.Extra
			},
		},
		{
			name:   "batch create",
			method: http.MethodPost,
			path:   "/providers/batch",
			body:   `{"providers":[{"name":"provider","platform":"openai","type":"apikey","credentials":{"api_key":"test"},` + malformedExtra + `}]}`,
			mount: func(router *gin.Engine, handler *ManagementHandler) {
				router.POST("/providers/batch", handler.BatchCreate)
			},
			capturedExtra: func(stub *managementMutationFixture) map[string]any {
				require.Len(t, stub.createdProviders, 1)
				return stub.createdProviders[0].Extra
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newManagementMutationFixture()
			handler := newMutationHandler(stub, nil)
			router := gin.New()
			tt.mount(router, handler)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			request.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			extra := tt.capturedExtra(stub)
			require.NotContains(t, extra, deprecatedLongContextBillingExtraKey)
			require.Equal(t, "value", extra["preserved"])
		})
	}
}

func TestProviderUpdateDeprecatedOnlyIsNormalizedToNoExtraUpdate(t *testing.T) {
	stub := newManagementMutationFixture()
	handler := newMutationHandler(stub, nil)
	router := gin.New()
	router.PUT("/providers/:id", handler.Update)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/providers/1", bytes.NewBufferString(
		`{"extra":{"openai_long_context_billing_enabled":{"malformed":true}}}`,
	))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.NotNil(t, stub.updateProviderInput)
	require.Nil(t, stub.updateProviderInput.Extra)
}

func TestApplyOAuthCredentialsDiscardsDeprecatedLongContextBillingExtra(t *testing.T) {
	stub := newManagementMutationFixture()
	stub.providers = []provider.Record{{
		ID:       1,
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeOAuth,
	}}
	handler := newMutationHandler(stub, nil)
	router := gin.New()
	router.POST("/providers/:id/apply-oauth-credentials", handler.ApplyOAuthCredentials)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/providers/1/apply-oauth-credentials", bytes.NewBufferString(
		`{"type":"oauth","credentials":{"access_token":"new-token"},"extra":{"openai_long_context_billing_enabled":1,"preserved":"value"}}`,
	))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.NotNil(t, stub.updateProviderInput)
	require.Len(t, stub.updateExtraCalls, 1)
	require.NotContains(t, stub.updateExtraCalls[0], deprecatedLongContextBillingExtraKey)
	require.Equal(t, "value", stub.updateExtraCalls[0]["preserved"])
}

const deprecatedLongContextBillingExtraKey = "openai_long_context_billing_enabled"
