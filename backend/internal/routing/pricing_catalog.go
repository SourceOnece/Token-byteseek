package routing

import (
	"time"

	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
)

type ModelPriceReader interface {
	GetModelPricing(string) (*ModelPricing, error)
}

// PricingCatalog 提供现有价格目录的查询和更新入口，不构建第二份目录缓存。
type PricingCatalog struct {
	Snapshot func() DefaultPricingSnapshot
	Update   func() error
	Prices   ModelPriceReader
}

func (c *PricingCatalog) DefaultPricing(model string) (*ModelPricing, error) {
	return c.Prices.GetModelPricing(model)
}

// DefaultPricingSnapshot 固定一次查询的模型名、价格和更新时间。
type DefaultPricingSnapshot struct {
	Version   string
	LastError string
	Prices    []pricing.DefaultModelPrice
	UpdatedAt time.Time
}
