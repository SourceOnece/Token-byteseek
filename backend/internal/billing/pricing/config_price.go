package pricing

// ApplyConfigPrice 在独立副本上应用价卡覆盖，保留 nil 图片价和倍率语义。
func ApplyConfigPrice(pricing *ModelPricing, configPricing *ModelPricingEntry) *ModelPricing {
	if configPricing == nil {
		return pricing
	}
	// 防止修改 目录中的共享价格
	cloned := *pricing
	pricing = &cloned
	ApplyConfigTokenPriceOverrides(pricing, configPricing)
	multiplier, configured := NormalizedPriceMultiplier(configPricing)
	if configured {
		pricing = MultiplyModelPricing(pricing, multiplier)
	}
	ApplyConfigFastModeMultiplier(pricing, configPricing)
	ApplyConfigFlexMultiplier(pricing, configPricing)
	if configPricing.MaxReasoningEffortMultiplier != nil {
		pricing.MaxReasoningEffortMultiplier = configPricing.MaxReasoningEffortMultiplier
	}
	return pricing
}
