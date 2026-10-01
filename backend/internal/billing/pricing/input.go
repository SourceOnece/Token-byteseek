package pricing

import (
	"time"
)

// CostInput 只携带本次计算的值快照，调用方决定数据来源和取时点。
type CostInput struct {
	ModelPricingAt      time.Time
	ModelTimeLocation   *time.Location
	Model               string
	Tokens              UsageTokens
	RequestCount        int
	UsageUnits          float64
	SizeTier            string
	RateMultiplier      float64
	PricingAt           time.Time
	ServiceTier         string
	ReasoningEffort     string
	TimePricingLocation *time.Location
}

// ResolvedTimeMultiplier 只计算显式 Location 下的分时倍率。
func ResolvedTimeMultiplier(resolved *ResolvedPricing, at time.Time, location *time.Location) float64 {
	if resolved == nil || resolved.Mode != BillingModeToken || resolved.ConfigPricing == nil {
		return 1
	}
	return resolved.ConfigPricing.TimePricing.MultiplierAt(at, location)
}
