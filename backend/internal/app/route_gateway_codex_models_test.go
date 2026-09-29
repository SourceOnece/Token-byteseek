package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/protocol"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

// codexModelsRemovalProviderRepo 提供仅含 API Key 提供商的分组模型数据。
type codexModelsRemovalProviderRepo struct {
	modelHTTPProviderRows
	providers []provider.Record
}

func (r *codexModelsRemovalProviderRepo) ListSchedulableByGroupID(context.Context, int64) ([]provider.Record, error) {
	return append([]provider.Record(nil), r.providers...), nil
}

// 带 client_version 的模型请求应继续返回纯 API Key 分组的本地模型列表。
func TestGatewayRoutesModelsWithClientVersionUsesLocalList(t *testing.T) {
	repo := &codexModelsRemovalProviderRepo{
		providers: []provider.Record{
			{
				ID:       1,
				Platform: capability.PlatformOpenAI,
				Type:     capability.ProviderTypeAPIKey,
				Credentials: map[string]any{
					"api_key":         "sk-test",
					"model_whitelist": []string{"local-api-key-model"},
					"model_mapping": map[string]any{
						"local-api-key-model": "local-api-key-model",
					},
				},
			},
		},
	}
	router := newGatewayRoutesTestRouterWithGroup(&config.Config{}, &routing.Group{
		ID: 1, AllowedProtocols: []protocol.ProtocolID{protocol.ProtocolAnthropicMessages, protocol.ProtocolOpenAIResponses, protocol.ProtocolOpenAIChatCompletions},
	}, newGatewayModelsHandlerForTest(repo))
	paths := []string{
		"/v1/models?client_version=0.144.0",
		"/models?client_version=0.144.0",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			require.Equal(t, http.StatusOK, recorder.Code)
			require.NotContains(t, recorder.Body.String(), "No available OpenAI OAuth providers")
			var response struct {
				Object string `json:"object"`
				Data   []struct {
					ID string `json:"id"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			require.Equal(t, "list", response.Object)
			require.Len(t, response.Data, 1)
			require.Equal(t, "local-api-key-model", response.Data[0].ID)
		})
	}
}

// Codex manifest 路由应被移除，已有 Responses 兼容路由仍需保留。
func TestGatewayRoutesCodexModelsManifestPathIsRemoved(t *testing.T) {
	router := newGatewayRoutesTestRouter(capability.PlatformOpenAI)

	req := httptest.NewRequest(http.MethodGet, "/backend-api/codex/models?client_version=0.144.0", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusNotFound, recorder.Code)

	// 合法 Live call 动态段仍需进入 Sideband handler，而不是被旧路由守卫误判为 404。
	req = httptest.NewRequest(http.MethodGet, "/backend-api/codex/call_test", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.NotEqual(t, http.StatusNotFound, recorder.Code)

	registered := make(map[string]string)
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = route.Handler
	}
	require.NotEmpty(t, registered[http.MethodPost+" /backend-api/codex/responses"])
	require.Empty(t, registered[http.MethodGet+" /backend-api/codex/models"])
	require.Equal(t, registered[http.MethodGet+" /v1/models"], registered[http.MethodGet+" /models"])
}

func TestGatewayRoutesRetrieveModelAliasesAndSlashIDs(t *testing.T) {
	repo := &codexModelsRemovalProviderRepo{providers: []provider.Record{{ID: 1, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey, Credentials: map[string]any{
		"model_whitelist": []any{"vendor/model"},
	}}}}
	router := newGatewayRoutesTestRouterWithGroup(&config.Config{}, &routing.Group{ID: 1, AllowedProtocols: []protocol.ProtocolID{protocol.ProtocolOpenAIResponses, protocol.ProtocolOpenAIChatCompletions}}, newGatewayModelsHandlerForTest(repo))
	for _, root := range []string{"/v1/models/", "/models/"} {
		for model, status := range map[string]int{"vendor/model": 200, "missing": 404} {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", root+model+"?client_version=0.144.0", nil))
			require.Equal(t, status, w.Code, w.Body.String())
		}
	}
}
