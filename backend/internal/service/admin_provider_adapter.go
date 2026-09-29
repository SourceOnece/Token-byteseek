package service

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// 旧仓储继续拥有调度通知和完整关联装配，原生管理用例不接触旧服务私有结构。
type providerAdminStoreAdapter struct{ AccountRepository }

func (s providerAdminStoreAdapter) ListShadowIDs(ctx context.Context, id int64) ([]int64, error) {
	shadows, err := s.ListShadowsByParent(ctx, id)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(shadows))
	for _, shadow := range shadows {
		ids = append(ids, shadow.ID)
	}
	return ids, nil
}

type providerRuntimeUnblocker struct{ AccountRuntimeBlocker }

func (b providerRuntimeUnblocker) ClearProviderSchedulingBlock(id int64) {
	b.ClearAccountSchedulingBlock(id)
}

func (s *adminServiceImpl) providerStateAdmin() *provider.Admin {
	var blocker provider.RuntimeUnblocker
	if s.runtimeBlocker != nil {
		blocker = providerRuntimeUnblocker{s.runtimeBlocker}
	}
	return provider.NewAdmin(providerAdminStoreAdapter{s.accountRepo}, blocker)
}
