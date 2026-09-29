package provider

import (
	"context"
	"reflect"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// 交错夹具模拟条件操作，真实 SQL 行锁/JSONB 条件另由集成测试验证。
func (r *refreshSuccessCooldownRepo) ClearRefreshCooldownIfUnchanged(ctx context.Context, v provider.RefreshCooldownVersion) (bool, error) {
	if !reflect.DeepEqual(provider.ObserveRefreshCooldown(r.current), v) {
		return false, nil
	}
	return true, r.ClearTempUnschedulable(ctx, v.ID)
}

func (r *tokenRefreshCandidateRepo) ClearRefreshCooldownIfUnchanged(_ context.Context, v provider.RefreshCooldownVersion) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.providers {
		if r.providers[i].ID == v.ID {
			if !reflect.DeepEqual(provider.ObserveRefreshCooldown(&r.providers[i]), v) {
				return false, nil
			}
			r.clearTempCalls++
			r.providers[i].TempUnschedulableUntil = nil
			r.providers[i].TempUnschedulableReason = ""
			return true, nil
		}
	}
	return false, nil
}
