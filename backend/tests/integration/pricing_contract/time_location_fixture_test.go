//go:build unit

package pricingcontract

import (
	"time"

	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	billingadapter "github.com/TokenFlux/TokenRouter/internal/billing/provider"
)

// 合同测试只提供显式时区，校验与倍率计算使用实际定价实现。
func contractTimeLocation(value *pricing.TimePricingConfig) *time.Location {
	if value == nil {
		return nil
	}
	location, _ := billingadapter.LoadPricingLocation(value.Timezone)
	return location
}
