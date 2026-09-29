package app

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	"github.com/TokenFlux/TokenRouter/internal/routing"
)

// routingGroupProviders 投影提供商存储；平台默认目录通过按需查询端口提供。
type routingGroupProviders struct {
	Store    *providerpostgres.ProviderStore
	Defaults provider.ModelMappingDefaults
}

func (r routingGroupProviders) GetByIDs(ctx context.Context, ids []int64) ([]routing.GroupProvider, error) {
	values, err := r.Store.GetByIDs(ctx, ids)
	// 原批量入口将缺省结果投影为空数组，保留与列表入口的 nil 差异。
	out := make([]routing.GroupProvider, len(values))
	for i, v := range values {
		out[i] = r.project(v)
	}
	return out, err
}

func (r routingGroupProviders) ListSchedulableByGroupID(ctx context.Context, id int64) ([]routing.GroupProvider, error) {
	values, err := r.Store.ListSchedulableByGroupID(ctx, id)
	if values == nil {
		return nil, err
	}
	out := make([]routing.GroupProvider, len(values))
	for i := range values {
		out[i] = r.project(&values[i])
	}
	return out, err
}

func (r routingGroupProviders) project(v *provider.Record) routing.GroupProvider {
	return routing.GroupProvider{ID: v.ID, Platform: v.Platform, Type: v.Type, Models: v.GetConfiguredRequestModels(r.Defaults)}
}
