package pricing

import (
	"fmt"
	"math"
)

// ConfiguredImageUnitPrice 保持显式零价与未定价之间的区别。
func ConfiguredImageUnitPrice(resolved *ResolvedPricing, size string) (float64, bool) {
	if resolved == nil || resolved.Mode != BillingModeImage && resolved.Mode != BillingModePerRequest {
		return 0, false
	}
	return ResolveRequestUnitPrice(resolved, size, nil)
}

// ValidateImageUnitPrice 保留旧固定单张价的失败语义。
func ValidateImageUnitPrice(price float64) (float64, error) {
	if math.IsNaN(price) || math.IsInf(price, 0) || price < 0 {
		return 0, fmt.Errorf("invalid image unit price: %w", ErrModelPricingUnavailable)
	}
	return price, nil
}

// ResolveImageUnitPrice 仅为图片模式或无按次价卡的请求补目录单价；按次缺价不借用图片价。
func ResolveImageUnitPrice(resolved *ResolvedPricing, catalog *CatalogModelPricing, size string) (float64, error) {
	price, found := ConfiguredImageUnitPrice(resolved, size)
	if !found && (resolved == nil || resolved.Mode != BillingModePerRequest) {
		price, found = DefaultImagePrice(catalog, size)
	}
	if !found {
		return 0, ErrModelPricingUnavailable
	}
	return ValidateImageUnitPrice(price)
}
