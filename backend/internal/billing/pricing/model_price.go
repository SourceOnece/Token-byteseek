package pricing

import (
	"fmt"
	"strings"
)

// ResolveModelPricing 获取模型价格配置
func ResolveModelPricing(model string, catalogPrice *CatalogModelPricing, prices map[string]*ModelPricing, policy ModelPolicy) (*ModelPricing, bool, error) {
	// 标准化模型名称（转小写）
	model = strings.ToLower(model)
	if catalogPrice != nil && (catalogPrice.Source == "unpriced" || catalogPrice.Source == "models.dev" && catalogPrice.TokenPricingAbsent) {
		return nil, false, fmt.Errorf("%w for model: %s", ErrModelPricingUnavailable, model)
	}

	// 1. 优先从动态价格服务获取
	if catalogPrice != nil {
		catalogPricing := catalogPrice
		// 仅有图片价、无 token 价的条目（例如 Imagen 模型）不能用于
		// token 计费：直接返回会把 token 流量按 $0 计费。跳过后走 fallback，
		// 无 fallback 则 fail-closed（ErrModelPricingUnavailable）。
		// 图片计费路径（getDefaultImagePrice / getImageUnitPrice）直接读
		// 目录读取接口，不受影响。
		if catalogPricing != nil && catalogPricing.TokenPricingAbsent {
			catalogPricing = nil
		}
		if catalogPricing != nil {
			inclusiveThreshold := strings.EqualFold(catalogPricing.Provider, "xai")
			var contextPrices []ContextModelPrice
			for _, tier := range catalogPricing.ContextPrices {
				if tier.Pricing == nil {
					continue
				}
				price, _, err := ResolveModelPricing(model, tier.Pricing, nil, policy)
				if err == nil {
					threshold := tier.Threshold
					// 内部阶梯统一使用 (min,max]；xAI 达到阈值即切档，先转换成前一档的最大 token 数。
					if inclusiveThreshold && threshold > 0 {
						threshold--
					}
					contextPrices = append(contextPrices, ContextModelPrice{Threshold: threshold, Pricing: price})
				}
			}
			// models.dev 明确给出 1h 单价时按 TTL 拆分，显式零价也有效。
			// 旧格式沿用 1h 高于 5m 的兼容判断。
			price5m := catalogPricing.CacheCreationInputTokenCost
			price1h := catalogPricing.CacheCreationInputTokenCostAbove1hr
			enableBreakdown := price1h > 0 && price1h > price5m || catalogPricing.Source == "models.dev" && catalogPricing.CacheCreation1hPricePresent
			return ApplyModelSpecificPricingPolicy(model, &ModelPricing{
				CatalogSource:                      catalogPricing.Source,
				CacheCreationPriceExplicit:         catalogPricing.Source != "" && catalogPricing.CacheCreationPricePresent && catalogPricing.CacheCreationInputTokenCost == 0,
				PriorityInputPresent:               catalogPricing.PriorityInputPresent,
				PriorityOutputPresent:              catalogPricing.PriorityOutputPresent,
				PriorityCacheReadPresent:           catalogPricing.PriorityCacheReadPresent,
				PriorityCacheWritePresent:          catalogPricing.PriorityCacheWritePresent,
				ContextPrices:                      contextPrices,
				InputPricePerToken:                 catalogPricing.InputCostPerToken,
				InputPricePerTokenPriority:         catalogPricing.InputCostPerTokenPriority,
				OutputPricePerToken:                catalogPricing.OutputCostPerToken,
				OutputPricePerTokenPriority:        catalogPricing.OutputCostPerTokenPriority,
				CacheCreationPricePerToken:         catalogPricing.CacheCreationInputTokenCost,
				CacheCreationPricePerTokenPriority: catalogPricing.CacheCreationInputTokenCostPriority,
				CacheReadPricePerToken:             catalogPricing.CacheReadInputTokenCost,
				CacheReadPricePerTokenPriority:     catalogPricing.CacheReadInputTokenCostPriority,
				CacheCreation5mPrice:               price5m,
				CacheCreation1hPrice:               price1h,
				SupportsCacheBreakdown:             enableBreakdown,
				SupportsServiceTier:                catalogPricing.SupportsServiceTier,
				// xAI 的目录语义是达到阈值即进入高档，其他提供商保持严格大于。
				LongContextThresholdInclusive: inclusiveThreshold,
				LongContextInputThreshold:     catalogPricing.LongContextInputTokenThreshold,
				LongContextInputMultiplier:    catalogPricing.LongContextInputCostMultiplier,
				LongContextOutputMultiplier:   catalogPricing.LongContextOutputCostMultiplier,
				ImageInputPricePerToken:       catalogPricing.InputCostPerImageToken,
				ImageCacheReadPricePerToken:   catalogPricing.CacheReadInputImageTokenCost,
				ImageOutputPricePerToken:      catalogPricing.OutputCostPerImageToken,
				ImageOutputPriceExplicit:      catalogPricing.ImageOutputPricePresent,
				MaxReasoningEffortMultiplier:  DefaultMaxReasoningEffortMultiplier(model),
			}, policy), false, nil
		}
	}

	// 2. 使用硬编码回退价格
	fallback := LookupFallbackPrice(prices, model)
	if fallback != nil {

		cloned := *fallback
		if cloned.MaxReasoningEffortMultiplier == nil {
			cloned.MaxReasoningEffortMultiplier = DefaultMaxReasoningEffortMultiplier(model)
		}
		return ApplyModelSpecificPricingPolicy(model, &cloned, policy), true, nil
	}

	return nil, false, fmt.Errorf("%w for model: %s", ErrModelPricingUnavailable, model)
}
