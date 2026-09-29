package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/grok"
)

// NewGrokQuotaView 将供应商档位解释绑定到提供商展示规则，不持有缓存或客户端。
func NewGrokQuotaView() *provider.GrokQuotaView {
	return &provider.GrokQuotaView{
		FreeTokenLimit:      grok.GrokFreeRolling24hTokenLimit,
		NeedsReauth:         provider.GrokNeedsReauth,
		JWTSubscriptionTier: grok.SubscriptionTierFromJWT,
		CanonicalPlan:       grok.CanonicalGrokPlan,
		ParseTime:           provider.ParseUsageTime,
	}
}
