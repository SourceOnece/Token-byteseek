package pricing

// LookupFallbackPrice 只查询完整型号的静态价格。
func LookupFallbackPrice(prices map[string]*ModelPricing, model string) *ModelPricing {
	return prices[NormalizePriceModelName(model)]
}
