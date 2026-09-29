package billing_test

import (
	"context"
	"testing"
	"time"

	billingtestkit "github.com/TokenFlux/TokenRouter/internal/billing/testkit"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	billingadapter "github.com/TokenFlux/TokenRouter/internal/billing/provider"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	routingtestkit "github.com/TokenFlux/TokenRouter/internal/routing/testkit"
	xai "github.com/TokenFlux/TokenRouter/internal/upstream/grok"

	"github.com/stretchr/testify/require"
)

// 新目录别名在有无内置价时均先采用共享价格配置价，不能以目录回退跳过运营者的显式配置。
func TestResolveCatalogAliasesPreserveConfigPricing(t *testing.T) {
	previous := xai.RuntimeModelMappingOptions()
	t.Cleanup(func() { xai.SetRuntimeModelMappingOptions(previous) })
	xai.SetRuntimeModelMappingOptions(xai.ModelMappingOptions{DefaultText: "X-AI/GROK-4.6"})
	for _, tc := range []struct{ platform, base, alias string }{
		{capability.PlatformGemini, "gemini-3.8-flash", "gemini-3.8-flash-tiered"},
		{capability.PlatformGemini, "gemini-3.7-flash", "models/gemini-3.7-flash-medium"},
		{capability.PlatformGrok, "grok-4.6", "grok-4.6-latest"},
		{capability.PlatformGrok, "grok-4.6", "grok-latest"},
		{capability.PlatformOpenAI, "gpt-5.6-luna", "gpt-5.6-luna-high"},
		{capability.PlatformAnthropic, "claude-opus-4-6", "claude-opus-4.6"},
	} {
		t.Run(tc.alias, func(t *testing.T) {
			for _, hasCatalog := range []bool{true, false} {
				groupID := int64(998)
				pricingConfigPrice := 9e-6
				configPricing := routingtestkit.Configuration{ID: 998, Status: billing.StatusActive, GroupIDs: []int64{groupID}, ModelPricing: []routing.ModelPricingEntry{{
					Models: []string{tc.base}, BillingMode: routing.BillingModeToken, InputPrice: &pricingConfigPrice,
				}}}
				repository := &routingtestkit.ConfigRows{Values: []routingtestkit.Configuration{configPricing}, Platforms: map[int64]string{groupID: tc.platform}}
				pricingConfigs := routingtestkit.NewPricingConfigService(repository, nil, routing.PricingConfigOptions{Now: time.Now, LoadLocation: billingadapter.LoadPricingLocation})
				var catalog *billingadapter.PricingService
				if hasCatalog {
					catalog = newCatalogFixture(catalogFixture{pricingData: map[string]*pricing.LiteLLMModelPricing{
						tc.base: {Mode: "chat", InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6},
					}})
				}
				resolver := billingtestkit.PriceResolver(pricingConfigs, newCalculator(&config.Config{}, catalog))
				input := billing.PricingInput{Model: tc.alias, GroupID: &groupID}
				resolved := resolver.Resolve(context.Background(), input)
				require.Equal(t, pricing.PricingSourceConfig, resolved.Source, "catalog=%v", hasCatalog)
				require.InDelta(t, pricingConfigPrice, resolved.BasePricing.InputPricePerToken, 1e-12)
				// Claude 分隔符写法在共享价格配置层属于同一个定价键，不能重复配置独立价卡。
				if pricing.NormalizePriceModelName(tc.base) == pricing.NormalizePriceModelName(tc.alias) {
					continue
				}

				// 完整请求名的独立价卡仍然优先，显式零价也不能被基础名价格覆盖。
				zero := 0.0
				configPricing.ModelPricing = append(configPricing.ModelPricing, routing.ModelPricingEntry{
					Models: []string{tc.alias}, BillingMode: routing.BillingModeToken, InputPrice: &zero,
				})
				repository.Values = []routingtestkit.Configuration{configPricing}
				pricingConfigs.InvalidateCache()
				resolved = resolver.Resolve(context.Background(), input)
				require.True(t, resolved.HasEffectivePricing())
				require.Zero(t, resolved.BasePricing.InputPricePerToken)
				base := resolver.Resolve(context.Background(), billing.PricingInput{Model: tc.base, GroupID: &groupID})
				require.InDelta(t, pricingConfigPrice, base.BasePricing.InputPricePerToken, 1e-12)
			}
		})
	}
}

// 同一共享价表中的基础模型价可用于对应别名，提供商平台不形成额外价格边界。
func TestResolveCatalogAliasesUseUnifiedPricingConfig(t *testing.T) {
	groupID := int64(999)
	price := 9e-6

	pricingConfigs := routingtestkit.ModelConfigFromData(routingtestkit.ModelConfigDataFromRows([]routingtestkit.Configuration{{
		ID: 999, Status: billing.StatusActive, GroupIDs: []int64{groupID}, ModelPricing: []routing.ModelPricingEntry{
			{Models: []string{"gemini-3.8-flash"}, BillingMode: routing.BillingModeToken, InputPrice: &price},
			{Models: []string{"gemini-3.7-flash"}, BillingMode: routing.BillingModeToken, InputPrice: &price},
		},
	}}, map[int64]string{groupID: capability.PlatformGemini}))
	catalog := newCatalogFixture(catalogFixture{pricingData: map[string]*pricing.LiteLLMModelPricing{
		"gemini-3.8-flash": {Mode: "chat", InputCostPerToken: 1e-6},
	}})
	resolver := billingtestkit.PriceResolver(pricingConfigs, newCalculator(&config.Config{}, catalog))
	resolved := resolver.Resolve(context.Background(), billing.PricingInput{Model: "gemini-3.8-flash-tiered", GroupID: &groupID})
	require.True(t, resolved.HasEffectivePricing())
	require.InDelta(t, price, resolved.BasePricing.InputPricePerToken, 1e-12)
}

// 两类价卡共用身份候选，空完整名条目不遮蔽基础价，完整名零价和通配价仍优先。
func TestGroupAndPricingCatalogAliasPrecedence(t *testing.T) {
	for _, tc := range []struct{ platform, base, alias string }{
		{capability.PlatformOpenAI, "gpt-5.6-luna", "gpt-5.6-luna-high"},
		{capability.PlatformGemini, "gemini-3.8-flash", "models/gemini-3.8-flash-tiered"},
		{capability.PlatformGrok, "grok-4.6", "grok-4.6-latest"},
		{capability.PlatformAnthropic, "claude-opus-4-6", "claude-opus-4.6"},
	} {
		t.Run(tc.alias, func(t *testing.T) {
			price, zero := 9e-6, 0.0
			card := routing.ModelPricingEntry{Models: []string{tc.base}, InputPrice: &price}
			for _, scope := range []string{pricing.PricingSourceConfig} {
				group := &routing.Group{ID: 990}

				repository := &routingtestkit.ConfigRows{Platforms: map[int64]string{group.ID: tc.platform}}
				pricingConfigs := routingtestkit.NewPricingConfigService(repository, nil, routing.PricingConfigOptions{Now: time.Now, LoadLocation: billingadapter.LoadPricingLocation})
				resolver := billingtestkit.PriceResolver(pricingConfigs, newCalculator(&config.Config{}, nil))
				resolve := func(cards []routing.ModelPricingEntry) *pricing.ResolvedPricing {
					configPricing := routingtestkit.Configuration{ID: 990, Status: billing.StatusActive, GroupIDs: []int64{group.ID}}
					configPricing.ModelPricing = cards
					repository.Values = []routingtestkit.Configuration{configPricing}
					pricingConfigs.InvalidateCache()
					return resolver.Resolve(context.Background(), billing.PricingInput{Model: tc.alias, GroupID: &group.ID})
				}
				base := resolve([]routing.ModelPricingEntry{card})
				require.Equal(t, scope, base.Source)
				require.Equal(t, price, base.BasePricing.InputPricePerToken)
				if pricing.NormalizePriceModelName(tc.base) == pricing.NormalizePriceModelName(tc.alias) {
					continue
				}
				exact := routing.ModelPricingEntry{Models: []string{tc.alias}}
				require.Equal(t, price, resolve([]routing.ModelPricingEntry{exact, card}).BasePricing.InputPricePerToken)
				exact.InputPrice = &zero
				require.Zero(t, resolve([]routing.ModelPricingEntry{card, exact}).BasePricing.InputPricePerToken)
				wildcard := routing.ModelPricingEntry{Models: []string{tc.alias + "*"}, InputPrice: &zero}
				require.Zero(t, resolve([]routing.ModelPricingEntry{card, wildcard}).BasePricing.InputPricePerToken)
			}
		})
	}
}
