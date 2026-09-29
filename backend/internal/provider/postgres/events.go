package postgres

import (
	"context"

	postgresinfra "github.com/TokenFlux/TokenRouter/internal/infra/postgres"
)

// ProviderEvent 表达存储写入产生的技术变更，实际 outbox 名称与编码由装配提供。
type ProviderEvent uint8

const (
	ProviderChanged ProviderEvent = iota
	ProviderGroupsChanged
	ProviderLastUsed
	ProviderBulkChanged
)

// ProviderEvents 统一原 outbox 编码及提交后的缓存发布，不在提供商存储复制调度缓存规则。
type ProviderEvents interface {
	Name(ProviderEvent) string
	Write(context.Context, postgresinfra.Executor, ProviderEvent, *int64, *int64, any) error
	GroupPayload([]int64) any
	SyncOne(context.Context, int64)
	SyncMany(context.Context, []int64)
	Drop(context.Context, int64)
}

func (r *ProviderStore) enqueue(ctx context.Context, exec postgresinfra.Executor, event ProviderEvent, providerID, groupID *int64, payload any) error {
	if r.options.Events == nil {
		return nil
	}
	return r.options.Events.Write(ctx, exec, event, providerID, groupID, payload)
}

func (r *ProviderStore) groupPayload(ids []int64) any {
	if r.options.Events == nil {
		return nil
	}
	return r.options.Events.GroupPayload(ids)
}

func (r *ProviderStore) eventName(event ProviderEvent) string { return r.options.Events.Name(event) }

func (r *ProviderStore) afterChanges(ctx context.Context, ids []int64) {
	if r.options.Events != nil {
		r.options.Events.SyncMany(ctx, ids)
	}
}

func (r *ProviderStore) dropSnapshot(ctx context.Context, id int64) {
	if r.options.Events != nil {
		r.options.Events.Drop(ctx, id)
	}
}
