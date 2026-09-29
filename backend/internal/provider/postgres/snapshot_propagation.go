package postgres

import (
	"context"
	"time"
)

// afterChangeDetached 在请求取消后仍以短超时传播最新提供商快照。
func (r *ProviderStore) afterChangeDetached(ctx context.Context, providerID int64) {
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	propagationCtx, cancel := context.WithTimeout(base, 2*time.Second)
	defer cancel()
	r.afterChange(propagationCtx, providerID)
}
