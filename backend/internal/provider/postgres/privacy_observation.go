package postgres

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// UpdatePrivacyModeIfUnchanged 在原 Extra/outbox 事务内比较请求身份，不能写到管理员更换后的提供商。
func (r *ProviderStore) UpdatePrivacyModeIfUnchanged(ctx context.Context, v provider.UsageObservationVersion, mode string) (bool, error) {
	return r.UpdateUsageExtraIfUnchanged(ctx, v, map[string]any{"privacy_mode": mode})
}
