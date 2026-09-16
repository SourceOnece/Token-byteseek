package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware "github.com/TokenFlux/TokenRouter/internal/server/middleware"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayModelRetrieveUsesExactFilteredCatalogue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 1, Platform: service.PlatformOpenAI, ModelAllowlist: service.GroupModelAllowlist{Enabled: true, Models: []string{"public"}}}
	h := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{byGroup: map[int64][]service.Account{1: {
		{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Credentials: map[string]any{"model_whitelist": []any{"real", "hidden"}}},
	}}})
	key := &service.APIKey{Group: group, ModelMapping: map[string]string{"public": "real"}}
	call := func(model string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/v1/models", nil)
		if model != "" {
			c.Params = gin.Params{{Key: "model", Value: "/" + model}}
		}
		c.Set(string(middleware.ContextKeyAPIKey), key)
		h.Models(c)
		return w
	}
	list := call("")
	require.Equal(t, http.StatusOK, list.Code)
	var payload struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &payload))
	require.Len(t, payload.Data, 1)
	item := call("public")
	require.Equal(t, http.StatusOK, item.Code)
	var result map[string]any
	require.NoError(t, json.Unmarshal(item.Body.Bytes(), &result))
	require.Equal(t, payload.Data[0], result)
	for _, name := range []string{"real", "hidden", "PUBLIC", "unknown"} {
		denied := call(name)
		require.Equal(t, http.StatusNotFound, denied.Code, name)
		require.Contains(t, denied.Body.String(), "model_not_found")
	}
}

func TestGatewayModelRetrievePreservesCompositeSubscriptionBoundary(t *testing.T) {
	h := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{byGroup: map[int64][]service.Account{
		1: {{ID: 1, Platform: service.PlatformOpenAI, Credentials: map[string]any{"model_mapping": map[string]any{"vendor/model": "vendor/model"}}}},
		2: {{ID: 2, Platform: service.PlatformOpenAI, Credentials: map[string]any{"model_mapping": map[string]any{"other": "other"}}}},
	}})
	key := &service.APIKey{IsComposite: true, BillingMode: service.APIKeyBillingModeSubscription,
		User: &service.User{Status: service.StatusActive, AllowedGroups: []int64{1, 2}},
		CompositeGroups: []service.APIKeyCompositeGroup{
			{GroupID: 1, Prefix: "Allowed", Group: &service.Group{ID: 1, Platform: service.PlatformOpenAI, Status: service.StatusActive}},
			{GroupID: 2, Prefix: "Denied", Group: &service.Group{ID: 2, Platform: service.PlatformOpenAI, Status: service.StatusActive}},
		},
	}
	for model, status := range map[string]int{"Allowed/vendor/model": 200, "Denied/other": 404, "vendor/model": 404} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/v1/models/"+model, nil)
		c.Params = gin.Params{{Key: "model", Value: "/" + model}}
		c.Set(string(middleware.ContextKeyAPIKey), key)
		c.Set(string(middleware.ContextKeyAPIKeyBilling), &middleware.APIKeyBillingContext{
			Mode: service.APIKeyBillingModeSubscription, Available: true,
			Subscription: &service.UserSubscription{Plan: &service.SubscriptionPlan{GroupIDs: []int64{1}}},
		})
		h.Models(c)
		require.Equal(t, status, w.Code, w.Body.String())
		if status == 200 {
			require.Contains(t, w.Body.String(), model)
		}
	}
}
