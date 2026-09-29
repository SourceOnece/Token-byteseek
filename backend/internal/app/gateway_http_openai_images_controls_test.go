package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/selection"
	"github.com/TokenFlux/TokenRouter/internal/gateway/requeststate"
	"github.com/TokenFlux/TokenRouter/internal/protocol"
	"github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/routing/testkit"

	keyhttp "github.com/TokenFlux/TokenRouter/internal/apikey/httpapi"
	"github.com/TokenFlux/TokenRouter/internal/identity/httpapi/authctx"

	gatewaymedia "github.com/TokenFlux/TokenRouter/internal/gateway/media"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/gateway/admission"
	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"

	"github.com/TokenFlux/TokenRouter/internal/identity"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIGatewayHandlerImages_DisabledGroupRejectsBeforeScheduling(t *testing.T) {
	body := []byte(`{"model":"gpt-image-2","prompt":"draw","size":"1024x1024"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	groupID := int64(111)
	c.Set(string(keyhttp.ContextKeyAPIKey), &apikey.APIKey{
		ID:      222,
		GroupID: &groupID,
		Group: &routing.Group{
			ID:                   groupID,
			AllowImageGeneration: false,
		},
		User: &identity.User{ID: 333},
	})
	c.Set(string(authctx.ContextKeyUser), authctx.AuthSubject{UserID: 333, Concurrency: 1})

	h := newGatewayHTTPEndpoints(gatewayHTTPFixtureInput{
		Source:      &gatewayExecutionFixture{},
		Funding:     &admission.FundingAdmission{},
		Keys:        &apikey.APIKeyService{},
		Concurrency: gatewayhttp.NewConcurrencyHelper(&scheduler.ConcurrencyService{}, gatewayhttp.SSEPingFormatNone, 0), Availability: newExecutionAvailabilityForTest(nil, nil, nil), Choices: newEmptyCompatibleSelectionFixture(),
	})

	h.Images(c)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, "permission_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
	require.Contains(t, rec.Body.String(), gatewaymedia.ImageGenerationPermissionMessage)
}

// TestOpenAIGatewayHandlerImagesValidatesGroupMappedModel 验证同步 Images 入口在分组映射后校验模型族。
func TestOpenAIGatewayHandlerImagesValidatesGroupMappedModel(t *testing.T) {
	groupID := int64(112)
	pricingConfigService := newGatewayExecutionPricingConfigServiceForTest(groupID, capability.PlatformOpenAI, testkit.Configuration{
		ID:           112,
		Status:       billing.StatusActive,
		ModelMapping: map[string]string{"draw-alias": "gpt-image-1", "gpt-image-2": "gpt-5.4"},
	})

	tests := []struct {
		name       string
		model      string
		allowImage bool
		wantStatus int
		wantText   string
	}{
		{name: "普通别名映射为生图模型", model: "draw-alias", wantStatus: http.StatusForbidden, wantText: gatewaymedia.ImageGenerationPermissionMessage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`{"model":"` + tt.model + `","prompt":"draw"}`)
			req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req
			apiKey := &apikey.APIKey{
				ID:      223,
				GroupID: &groupID,
				Group: &routing.Group{
					ID:                   groupID,
					AllowImageGeneration: tt.allowImage,
				},
				User: &identity.User{ID: 334},
			}
			c.Set(string(keyhttp.ContextKeyAPIKey), apiKey)
			c.Set(string(authctx.ContextKeyUser), authctx.AuthSubject{UserID: 334, Concurrency: 1})

			newOpenAIImageChatRejectionHandlerWithPricingConfig(t, pricingConfigService).Images(c)

			require.Equal(t, tt.wantStatus, rec.Code)
			require.Contains(t, gjson.GetBytes(rec.Body.Bytes(), "error.message").String(), tt.wantText)
		})
	}
}

// 最终模型校验在候选阶段执行，避免入口把提供商别名误当成非图片模型。
func TestImagesGroupMappedTextModelIsRejectedByActualCandidate(t *testing.T) {
	groupID := int64(112)
	policies := newGatewayExecutionPricingConfigServiceForTest(groupID, "openai", testkit.Configuration{ID: 112, Status: billing.StatusActive, ModelMapping: map[string]string{"gpt-image-2": "gpt-5.4"}})
	provider := gatewayadapter.NewExecutionProvider(&provider.Record{ID: 1, Platform: "openai", Type: "apikey", Status: "active", Schedulable: true, GroupIDs: []int64{groupID}, Credentials: map[string]any{"model_whitelist": []string{"gpt-5.4"}}})
	store := &mixedHTTPProviders{values: []gatewayadapter.ExecutionProvider{*provider}}
	choices := selection.NewCompatible(selection.CompatibleDependencies{Reads: selection.Reads{Providers: store}, Shared: selection.Shared{GroupPolicies: policies}}, selection.DefaultOptions())
	ctx := requeststate.WithClientProtocol(requeststate.WithGroup(context.Background(), &routing.Group{ID: groupID, Hydrated: true, Status: "active", AllowedProtocols: []protocol.ProtocolID{protocol.ProtocolImagesGenerations}}), protocol.ProtocolImagesGenerations)
	selected, _, err := choices.SelectProviderWithSchedulerForImages(ctx, &groupID, "", "gpt-image-2", nil, gatewaymedia.ImageCapabilityNative)
	require.Error(t, err)
	require.Nil(t, selected)
}
