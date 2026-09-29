package catalogue_test

import (
	"context"
	"testing"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	xai "github.com/TokenFlux/TokenRouter/internal/upstream/grok"
	"github.com/stretchr/testify/require"
)

// 原默认目录与提供商显式别名断言直接组合原生解析器。
func TestGrokRequestableModelsExcludeBuiltinAliases(t *testing.T) {
	groupID := int64(4510)
	provider := providercore.Record{ID: 1, Platform: capability.PlatformGrok, Type: capability.ProviderTypeAPIKey, Credentials: map[string]any{}}
	repo := &modelsListProviderRepoStub{byGroup: map[int64][]providercore.Record{groupID: {provider}}}
	service := newCatalogueFixture(repo, nil, nil)

	result := service.ResolveRequestableModels(context.Background(), &groupID, capability.PlatformGrok)
	require.ElementsMatch(t, xai.DefaultModelIDs(), routing.RequestableModelIDs(result.Models))
	require.NotContains(t, routing.RequestableModelIDs(result.Models), "grok")
	require.NotContains(t, routing.RequestableModelIDs(result.Models), "grok-latest")

	provider.Credentials["model_mapping"] = map[string]any{"grok": "grok-4.3"}
	repo.byGroup = map[int64][]providercore.Record{groupID: {provider}}
	result = service.ResolveRequestableModels(context.Background(), &groupID, capability.PlatformGrok)
	require.Contains(t, routing.RequestableModelIDs(result.Models), "grok")
}
