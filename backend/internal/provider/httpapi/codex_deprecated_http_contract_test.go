package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCodexSessionImportDiscardsDeprecatedLongContextBillingExtra(t *testing.T) {
	stub := newCodexImportMemoryAdminService(nil)
	handler := NewCodexImportHandler(newCodexImportFixture(stub))
	router := gin.New()
	router.POST("/providers/import-codex-session", handler.ImportCodexSession)
	body, err := json.Marshal(provider.CodexSessionImportRequest{
		Content: buildCodexAccessToken(t, "workspace-1", "user-1", time.Now().Add(time.Hour)),
		Extra:   map[string]any{deprecatedLongContextBillingExtraKey: []bool{true}, "preserved": "value"},
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/providers/import-codex-session", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Len(t, stub.createdProviders, 1)
	require.NotContains(t, stub.createdProviders[0].Extra, deprecatedLongContextBillingExtraKey)
	require.Equal(t, "value", stub.createdProviders[0].Extra["preserved"])
}
