package pricing

import (
	"strings"
)

func GetDefaultGrokImagineImagePrice(model string, imageSize string) (float64, bool) {
	model = strings.ToLower(strings.TrimSpace(model))
	switch model {
	case "grok-imagine-image-2.0":
		return GetGrokImagineImageTierPrice(
			imageSize,
			DefaultGrokImagineImage20Price1K,
			DefaultGrokImagineImage20Price2K,
		), true
	case "grok-imagine-image-quality":
		return GetGrokImagineImageTierPrice(
			imageSize,
			DefaultGrokImagineImageQualityPrice1K,
			DefaultGrokImagineImageQualityPrice2K,
		), true
	case "grok-imagine-image":
		return GetGrokImagineImageTierPrice(
			imageSize,
			DefaultGrokImagineImagePrice1K,
			DefaultGrokImagineImagePrice2K,
		), true
	default:
		return 0, false
	}
}

func GetGrokImagineImageTierPrice(imageSize string, price1K float64, price2K float64) float64 {
	switch NormalizeImageBillingTierOrDefault(imageSize) {
	case ImageBillingSize1K:
		return price1K
	case ImageBillingSize2K, ImageBillingSize4K:
		return price2K
	default:
		return price2K
	}
}

func GetDefaultGrokImagineVideoPrice(model string, resolution string) (float64, bool) {
	model = strings.ToLower(strings.TrimSpace(model))
	switch model {
	case "grok-imagine-video-1.5":
		switch NormalizeVideoBillingResolutionOrDefault(resolution) {
		case VideoBillingResolution480P:
			return DefaultGrokImagineVideo15Price480P, true
		case VideoBillingResolution720P:
			return DefaultGrokImagineVideo15Price720P, true
		case VideoBillingResolution1080P:
			return DefaultGrokImagineVideo15Price1080P, true
		default:
			return DefaultGrokImagineVideo15Price480P, true
		}
	case "grok-imagine-video":
		switch NormalizeVideoBillingResolutionOrDefault(resolution) {
		case VideoBillingResolution480P:
			return DefaultGrokImagineVideoPrice480P, true
		case VideoBillingResolution720P, VideoBillingResolution1080P:
			return DefaultGrokImagineVideoPrice720P, true
		default:
			return DefaultGrokImagineVideoPrice480P, true
		}
	default:
		return 0, false
	}
}

// CalculateImageCost 对显式单张价格计算费用，保持乘法顺序与负倍率回退。
func CalculateImageCost(unitPrice float64, imageCount int, rateMultiplier float64) *CostBreakdown {
	if imageCount <= 0 {
		return &CostBreakdown{}
	}
	// 计算总费用
	totalCost := unitPrice * float64(imageCount)

	// 应用倍率（保存时强制 > 0；负数按 0 处理避免按 1x 误扣）
	if rateMultiplier < 0 {
		rateMultiplier = 0
	}
	actualCost := totalCost * rateMultiplier

	return &CostBreakdown{
		TotalCost:   totalCost,
		ActualCost:  actualCost,
		BillingMode: string(BillingModeImage),
	}
}

// CalculateVideoCost 对显式每秒价格计算费用，时长规则使用同一纯定义。
func CalculateVideoCost(perSecondPrice float64, videoCount, durationSeconds int, rateMultiplier float64) *CostBreakdown {
	if videoCount <= 0 {
		return &CostBreakdown{}
	}
	durationSeconds = NormalizeVideoBillingDurationSecondsOrDefault(durationSeconds)
	totalCost := perSecondPrice * float64(durationSeconds) * float64(videoCount)

	if rateMultiplier < 0 {
		rateMultiplier = 0
	}
	actualCost := totalCost * rateMultiplier

	return &CostBreakdown{
		TotalCost:   totalCost,
		ActualCost:  actualCost,
		BillingMode: string(BillingModeVideo),
	}
}

// HasImageUnitPrice 识别目录明确提供的按张价格，显式零价也有效。
func HasImageUnitPrice(catalogPrice *CatalogModelPricing) bool {
	return catalogPrice != nil && (catalogPrice.OutputCostPerImage > 0 ||
		catalogPrice.ImagePricePresent && catalogPrice.OutputCostPerImage == 0)
}

// DefaultImagePrice 对明确的目录报价应用尺寸倍率，布尔值区分显式零价与缺价。
func DefaultImagePrice(catalogPrice *CatalogModelPricing, imageSize string) (float64, bool) {
	if !HasImageUnitPrice(catalogPrice) {
		return 0, false
	}
	basePrice := catalogPrice.OutputCostPerImage

	// 2K 尺寸 1.5 倍，4K 尺寸翻倍
	if imageSize == "2K" {
		return basePrice * 1.5, true
	}
	if imageSize == "4K" {
		return basePrice * 2, true
	}

	return basePrice, true
}
