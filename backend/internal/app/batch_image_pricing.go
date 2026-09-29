package app

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/batchimage"
	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/routing"
)

// batchPricingGroups 只投影任务报价所需字段，读取仍由任务用例按需触发。
type batchPricingGroups struct {
	source   routing.GroupRepository
	settings *routing.PricingConfigService
}

func (r batchPricingGroups) GetByIDLite(ctx context.Context, id int64) (*batchimage.GroupView, error) {
	value, err := r.source.GetByIDLite(ctx, id)
	if value == nil {
		return nil, err
	}
	settings := r.settings.GetEffectiveBillingSettings(ctx, id)
	return &batchimage.GroupView{ID: value.ID, AllowBatchImageGeneration: value.AllowBatchImageGeneration, RateMultiplier: value.RateMultiplier, BatchImageDiscountMultiplier: settings.BatchImageDiscountMultiplier, BatchImageHoldMultiplier: settings.BatchImageHoldMultiplier}, err
}

func provideBatchPricing(resolver *billing.PriceResolver, groups routing.GroupRepository, configs *routing.PricingConfigService) *batchimage.Pricing {
	return &batchimage.Pricing{Resolver: resolver, GroupRepo: batchPricingGroups{groups, configs}}
}
