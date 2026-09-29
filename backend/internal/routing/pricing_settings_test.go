package routing

import (
	"context"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
	"github.com/stretchr/testify/require"
)

// 设置测试使用同一存储副本验证 CRUD、热缓存和关联变化。
type settingsRepository struct {
	PricingConfigRepository
	value *PricingConfig
}

func (r *settingsRepository) ExistsByName(context.Context, string) (bool, error) { return false, nil }

func (r *settingsRepository) GetGroupsInOtherPricingConfigs(context.Context, int64, []int64) ([]int64, error) {
	return nil, nil
}

func (r *settingsRepository) Create(_ context.Context, c *PricingConfig) error {
	c.ID = 1
	r.value = c.Clone()
	return nil
}

func (r *settingsRepository) GetByID(context.Context, int64) (*PricingConfig, error) {
	return r.value.Clone(), nil
}

func (r *settingsRepository) Update(_ context.Context, c *PricingConfig) error {
	r.value = c.Clone()
	return nil
}

func (r *settingsRepository) ListAll(context.Context) ([]PricingConfig, error) {
	if r.value == nil {
		return nil, nil
	}
	return []PricingConfig{*r.value.Clone()}, nil
}

func (r *settingsRepository) GetGroupIDs(context.Context, int64) ([]int64, error) {
	return append([]int64(nil), r.value.GroupIDs...), nil
}
func (r *settingsRepository) Delete(context.Context, int64) error { r.value = nil; return nil }

type settingsInvalidator struct{ groups []int64 }

func (i *settingsInvalidator) InvalidateAuthCacheByGroupID(_ context.Context, id int64) {
	i.groups = append(i.groups, id)
}

func TestSharedBillingSettingsCRUDAndCache(t *testing.T) {
	ctx := context.Background()
	repo := &settingsRepository{}
	invalidator := &settingsInvalidator{}
	svc := NewPricingConfigService(repo, invalidator)
	created, err := svc.Create(ctx, &CreatePricingConfigInput{Name: "shared", GroupIDs: []int64{7, 8}})
	require.NoError(t, err)
	require.Equal(t, pricing.DefaultBillingSettings(), created.BillingSettings)
	require.ElementsMatch(t, []int64{7, 8}, invalidator.groups)
	// 没有模型条目时，配置开关和显式零价仍独立生效。
	zero, enabled, disabled := 0.0, true, false
	_, err = svc.Update(ctx, created.ID, &UpdatePricingConfigInput{BillingSettingsPatch: BillingSettingsPatch{
		FreeOpenAIFast: &enabled, LongContextPricingEnabled: &disabled, WebSearchPricePerCall: PriceUpdate{Set: true, Value: &zero},
	}})
	require.NoError(t, err)
	for _, id := range []int64{7, 8} {
		got := svc.GetEffectiveBillingSettings(ctx, id)
		require.True(t, got.FreeOpenAIFast)
		require.False(t, got.LongContextPricingEnabled)
		require.NotNil(t, got.WebSearchPricePerCall)
		require.Zero(t, *got.WebSearchPricePerCall)
		*got.WebSearchPricePerCall = 99
	}
	require.Zero(t, *svc.GetEffectiveBillingSettings(ctx, 7).WebSearchPricePerCall)
	_, err = svc.Update(ctx, created.ID, &UpdatePricingConfigInput{BillingSettingsPatch: BillingSettingsPatch{WebSearchPricePerCall: PriceUpdate{Set: true}}})
	require.NoError(t, err)
	require.Nil(t, svc.GetEffectiveBillingSettings(ctx, 7).WebSearchPricePerCall)
	require.True(t, svc.GetEffectiveBillingSettings(ctx, 7).FreeOpenAIFast)
	invalidator.groups = nil
	members := []int64{8, 9}
	_, err = svc.Update(ctx, created.ID, &UpdatePricingConfigInput{GroupIDs: &members})
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{7, 8, 9}, invalidator.groups)
	require.Equal(t, pricing.DefaultBillingSettings(), svc.GetEffectiveBillingSettings(ctx, 7))
	require.True(t, svc.GetEffectiveBillingSettings(ctx, 9).FreeOpenAIFast)
	_, err = svc.Update(ctx, created.ID, &UpdatePricingConfigInput{Status: StatusDisabled})
	require.NoError(t, err)
	require.Equal(t, pricing.DefaultBillingSettings(), svc.GetEffectiveBillingSettings(ctx, 9))
	require.NoError(t, svc.Delete(ctx, created.ID))
	require.Equal(t, pricing.DefaultBillingSettings(), svc.GetEffectiveBillingSettings(ctx, 8))
}

func TestBillingSettingsValidateMergedInput(t *testing.T) {
	s := pricing.DefaultBillingSettings()
	enabled, start, end := true, "14:00", "18:00"
	peak, discount, hold := 0.0, 0.8, 0.9
	require.NoError(t, (BillingSettingsPatch{PeakRateEnabled: &enabled, PeakStart: &start, PeakEnd: &end, PeakRateMultiplier: &peak, BatchImageDiscountMultiplier: &discount, BatchImageHoldMultiplier: &hold}).Apply(&s))
	badEnd := "13:00"
	require.Error(t, (BillingSettingsPatch{PeakEnd: &badEnd}).Apply(&s))
	s = pricing.DefaultBillingSettings()
	require.Error(t, (BillingSettingsPatch{BatchImageDiscountMultiplier: &discount}).Apply(&s))
	negative := -1.0
	require.Error(t, (BillingSettingsPatch{SearchPricePer1k: PriceUpdate{Set: true, Value: &negative}}).Apply(&s))
}
