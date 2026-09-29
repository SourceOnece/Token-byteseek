package provider

import (
	"context"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/creative"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/stretchr/testify/require"
)

type creativeCatalogRows struct{ values []creative.CatalogProvider }

func (s creativeCatalogRows) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]creative.CatalogProvider, error) {
	return s.values, nil
}

// 透传只决定传输方式，生产目录和执行都保留显式模型映射。
func TestCreativeCatalogUsesExecutionModelPolicy(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		value := &provider.Record{
			Platform: creative.PlatformOpenAI, Type: "apikey", Status: "active", Schedulable: true,
			Credentials: map[string]any{"model_mapping": map[string]any{"gpt-image-1": "gpt-image-2"}},
			Extra:       map[string]any{"openai_passthrough": passthrough},
		}
		public := creative.Public{ProviderRepo: creativeCatalogRows{[]creative.CatalogProvider{CreativeCatalogProvider(value)}}}
		group := &creative.GroupView{ID: 12, Operations: creative.OperationsForGroup(false, nil), RoutingPolicy: routing.GroupRoutingPolicy{
			Enabled: true, RestrictModels: true, RestrictionModelSource: routing.BillingModelSourceUpstream,
			AllowedModels: []string{"gpt-image-2"},
		}}
		models, err := public.CreativeModelsForGroup(context.Background(), group)
		require.NoError(t, err)
		require.Equal(t, "gpt-image-2", models["gpt-image-1"])
	}
}
