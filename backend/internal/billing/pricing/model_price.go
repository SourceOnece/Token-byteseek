package pricing

import (
	"fmt"
	"strings"
)

// GetModelPricing 获取模型价格配置
func ResolveModelPricing(model string, catalogPrice *LiteLLMModelPricing, prices map[string]*ModelPricing, policy ModelPolicy) (*ModelPricing, bool, error) {
	// 标准化模型名称（转小写）
	model = strings.ToLower(model)
	if catalogPrice != nil && catalogPrice.Source == "unpriced" {
		return nil, false, fmt.Errorf("%w for model: %s", ErrModelPricingUnavailable, model)
	}

	// 1. 优先从动态价格服务获取
	if catalogPrice != nil {
		litellmPricing := catalogPrice
		// 仅有图片价、无 token 价的条目（如 LiteLLM 的 imagen 类模型）不能用于
		// token 计费：直接返回会把 token 流量按 $0 计费。跳过后走 fallback，
		// 无 fallback 则 fail-closed（ErrModelPricingUnavailable）。
		// 图片计费路径（getDefaultImagePrice / getImageUnitPrice）直接读
		// PricingService，不受影响。
		if litellmPricing != nil && litellmPricing.TokenPricingAbsent {
			litellmPricing = nil
		}
		if litellmPricing != nil {
			var contextPrices []ContextModelPrice
			for _, tier := range litellmPricing.ContextPrices {
				if tier.Pricing == nil {
					continue
				}
				price, _, err := ResolveModelPricing(model, tier.Pricing, nil, policy)
				if err == nil {
					contextPrices = append(contextPrices, ContextModelPrice{Threshold: tier.Threshold, Pricing: price})
				}
			}
			// models.dev 明确给出 1h 单价时按 TTL 拆分，显式零价也有效。
			// 旧格式沿用 1h 高于 5m 的兼容判断。
			price5m := litellmPricing.CacheCreationInputTokenCost
			price1h := litellmPricing.CacheCreationInputTokenCostAbove1hr
			enableBreakdown := price1h > 0 && price1h > price5m || litellmPricing.Source == "models.dev" && litellmPricing.CacheCreation1hPricePresent
			return ApplyModelSpecificPricingPolicy(model, &ModelPricing{
				CatalogSource:                      litellmPricing.Source,
				CacheCreationPriceExplicit:         litellmPricing.Source != "" && litellmPricing.CacheCreationPricePresent && litellmPricing.CacheCreationInputTokenCost == 0,
				PriorityInputPresent:               litellmPricing.PriorityInputPresent,
				PriorityOutputPresent:              litellmPricing.PriorityOutputPresent,
				PriorityCacheReadPresent:           litellmPricing.PriorityCacheReadPresent,
				PriorityCacheWritePresent:          litellmPricing.PriorityCacheWritePresent,
				ContextPrices:                      contextPrices,
				InputPricePerToken:                 litellmPricing.InputCostPerToken,
				InputPricePerTokenPriority:         litellmPricing.InputCostPerTokenPriority,
				OutputPricePerToken:                litellmPricing.OutputCostPerToken,
				OutputPricePerTokenPriority:        litellmPricing.OutputCostPerTokenPriority,
				CacheCreationPricePerToken:         litellmPricing.CacheCreationInputTokenCost,
				CacheCreationPricePerTokenPriority: litellmPricing.CacheCreationInputTokenCostPriority,
				CacheReadPricePerToken:             litellmPricing.CacheReadInputTokenCost,
				CacheReadPricePerTokenPriority:     litellmPricing.CacheReadInputTokenCostPriority,
				CacheCreation5mPrice:               price5m,
				CacheCreation1hPrice:               price1h,
				SupportsCacheBreakdown:             enableBreakdown,
				SupportsServiceTier:                litellmPricing.SupportsServiceTier,
				// xAI 的目录语义是达到阈值即进入高档，其他提供商保持严格大于。
				LongContextThresholdInclusive: strings.EqualFold(litellmPricing.LiteLLMProvider, "xai"),
				LongContextInputThreshold:     litellmPricing.LongContextInputTokenThreshold,
				LongContextInputMultiplier:    litellmPricing.LongContextInputCostMultiplier,
				LongContextOutputMultiplier:   litellmPricing.LongContextOutputCostMultiplier,
				ImageInputPricePerToken:       litellmPricing.InputCostPerImageToken,
				ImageCacheReadPricePerToken:   litellmPricing.CacheReadInputImageTokenCost,
				ImageOutputPricePerToken:      litellmPricing.OutputCostPerImageToken,
				MaxReasoningEffortMultiplier:  DefaultMaxReasoningEffortMultiplier(model),
			}, policy), false, nil
		}
	}

	// 2. 使用硬编码回退价格
	fallback := LookupFallbackPrice(prices, model, policy)
	if fallback != nil {

		cloned := *fallback
		if cloned.MaxReasoningEffortMultiplier == nil {
			cloned.MaxReasoningEffortMultiplier = DefaultMaxReasoningEffortMultiplier(model)
		}
		return ApplyModelSpecificPricingPolicy(model, &cloned, policy), true, nil
	}

	return nil, false, fmt.Errorf("%w for model: %s", ErrModelPricingUnavailable, model)
}
