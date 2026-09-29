// Grok 声明解析接入提供商纯规则，不持有缓存或凭据副本。
package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/grok"
)

func GrokTierRules() provider.GrokTierRules {
	return provider.GrokTierRules{
		SubscriptionTierFromJWT:   grok.SubscriptionTierFromJWT,
		NormalizeSubscriptionTier: grok.NormalizeSubscriptionTier,
		IsFreeRollingTokenLimit:   grok.IsGrokFreeRolling24hTokenLimit,
	}
}
