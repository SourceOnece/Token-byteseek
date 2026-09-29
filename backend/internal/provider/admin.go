package provider

import (
	"context"
	"fmt"
)

// AdminStateStore 只暴露本批迁移的状态管理能力；不把旧完整服务反向依赖引入原生模块。
type AdminStateStore interface {
	ListShadowIDs(context.Context, int64) ([]int64, error)
	Delete(context.Context, int64) error
	SetSchedulable(context.Context, int64, bool) error
	ClearError(context.Context, int64) error
	ClearRateLimit(context.Context, int64) error
	ClearAntigravityQuotaScopes(context.Context, int64) error
	ClearModelRateLimits(context.Context, int64) error
	ClearTempUnschedulable(context.Context, int64) error
}

type RuntimeUnblocker interface{ ClearProviderSchedulingBlock(int64) }

// Admin 先拥有状态管理，价格/分组关系由旧适配边界保留，后续再分批迁入。
type Admin struct {
	store     AdminStateStore
	unblocker RuntimeUnblocker
}

func NewAdmin(store AdminStateStore, unblocker RuntimeUnblocker) *Admin {
	return &Admin{store: store, unblocker: unblocker}
}

// DeleteProvider 保留母号删除前逐个删除影子的顺序及短路错误，避免留下持凭据母号已丢失的影子。
func (a *Admin) DeleteProvider(ctx context.Context, id int64) error {
	ids, err := a.store.ListShadowIDs(ctx, id)
	if err != nil {
		return fmt.Errorf("list spark shadows for cascade delete: %w", err)
	}
	for _, shadow := range ids {
		if err := a.store.Delete(ctx, shadow); err != nil {
			return fmt.Errorf("cascade delete spark shadow %d: %w", shadow, err)
		}
	}
	return a.store.Delete(ctx, id)
}

// ClearProviderError 持久清错全部成功后才解除运行时阻断；不直接开启总调度。
func (a *Admin) ClearProviderError(ctx context.Context, id int64) error {
	for _, clear := range []func(context.Context, int64) error{
		a.store.ClearError, a.store.ClearRateLimit, a.store.ClearAntigravityQuotaScopes,
		a.store.ClearModelRateLimits, a.store.ClearTempUnschedulable,
	} {
		if err := clear(ctx, id); err != nil {
			return err
		}
	}
	if a.unblocker != nil {
		a.unblocker.ClearProviderSchedulingBlock(id)
	}
	return nil
}

// SetProviderSchedulable 只修改管理员明确给定的总开关，不清票据、额度或质量结论。
func (a *Admin) SetProviderSchedulable(ctx context.Context, id int64, enabled bool) error {
	return a.store.SetSchedulable(ctx, id, enabled)
}
