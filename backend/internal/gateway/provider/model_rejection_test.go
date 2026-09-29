package provider

import (
	"testing"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
	"github.com/stretchr/testify/require"
)

func TestDefaultRequestModelIDsForPlatformQoder(t *testing.T) {
	require.Equal(t, qoder.DefaultRequestModelIDs(), DefaultRequestModels(capability.PlatformQoder))
}

func TestAvailableRequestModelsFromProvidersUsesQoderProviderSite(t *testing.T) {
	newProvider := func(id int64, site string) providercore.Record {
		return providercore.Record{
			ID:          id,
			Platform:    capability.PlatformQoder,
			Status:      providercore.StatusActive,
			Schedulable: true,
			Credentials: map[string]any{"site": site},
		}
	}

	cnModels := rejectedModelsForContract([]providercore.Record{newProvider(1, "cn")}, capability.PlatformQoder)
	require.ElementsMatch(t, qoder.DefaultRequestModelIDsForSite(qoder.SiteCN), cnModels)
	require.NotContains(t, cnModels, "claude-opus-4-6")

	globalModels := rejectedModelsForContract([]providercore.Record{newProvider(2, "global")}, capability.PlatformQoder)
	require.ElementsMatch(t, qoder.DefaultRequestModelIDsForSite(qoder.SiteGlobal), globalModels)
	require.NotContains(t, globalModels, "minimax-m2.7")

	mixedModels := rejectedModelsForContract([]providercore.Record{newProvider(3, "global"), newProvider(4, "cn")}, capability.PlatformQoder)
	require.ElementsMatch(t, qoder.DefaultRequestModelIDs(), mixedModels)
}

func TestAvailableRequestModelsFromProvidersFiltersConfiguredQoderModels(t *testing.T) {
	newProvider := func(id int64, site string, credentials map[string]any) providercore.Record {
		credentials["site"] = site
		return providercore.Record{
			ID:          id,
			Platform:    capability.PlatformQoder,
			Status:      providercore.StatusActive,
			Schedulable: true,
			Credentials: credentials,
		}
	}

	cnWhitelist := newProvider(11, "cn", map[string]any{
		"model_whitelist": []any{"claude-opus-4-6", "qwen3.6-flash"},
	})
	cnModels := rejectedModelsForContract([]providercore.Record{cnWhitelist}, capability.PlatformQoder)
	require.Equal(t, []string{"qwen3.6-flash"}, cnModels)

	cnMappingOverride := newProvider(12, "cn", map[string]any{
		"model_mapping": map[string]any{"claude-opus-4-6": "ultimate"},
	})
	overrideModels := rejectedModelsForContract([]providercore.Record{cnMappingOverride}, capability.PlatformQoder)
	require.ElementsMatch(t, qoder.DefaultRequestModelIDsForSite(qoder.SiteCN), overrideModels)
	require.NotContains(t, overrideModels, "claude-opus-4-6", "显式映射不能突破站点能力")

	globalWhitelist := newProvider(13, "global", map[string]any{
		"model_whitelist": []any{"claude-opus-4-6", "qwen3.6-flash"},
	})
	mixedModels := rejectedModelsForContract([]providercore.Record{cnWhitelist, globalWhitelist}, capability.PlatformQoder)
	require.ElementsMatch(t, []string{"claude-opus-4-6", "qwen3.6-flash"}, mixedModels)
}

// rejectedModelsForContract 只装配原生记录与 routing 规则，断言不经过旧提供商形状。
func rejectedModelsForContract(values []providercore.Record, platform string) []string {
	sources := make([]routing.ModelRejectionSource, len(values))
	for i := range values {
		sources[i] = ModelRejectionProvider(&values[i])
	}
	return routing.AvailableModelsForRejection(sources, platform)
}
