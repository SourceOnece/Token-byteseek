package provider_test

import (
	"context"
	"sync"
	"time"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
)

// providerServiceTestRepo 为提供商服务测试提供最小内存仓储。
type providerServiceTestRepo struct {
	providercore.AdminStore
	mu          sync.Mutex
	providers   map[int64]*providercore.Record
	updates     map[int64][]map[string]any
	bulkUpdates []providercore.ProviderBulkUpdate
}

func (r *providerServiceTestRepo) Create(_ context.Context, provider *providercore.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.providers == nil {
		r.providers = make(map[int64]*providercore.Record)
	}
	if provider.ID == 0 {
		provider.ID = int64(len(r.providers) + 1)
	}
	r.providers[provider.ID] = providercore.CloneRecord(provider)
	return nil
}

func (r *providerServiceTestRepo) Update(_ context.Context, provider *providercore.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.providers == nil {
		r.providers = make(map[int64]*providercore.Record)
	}
	r.providers[provider.ID] = providercore.CloneRecord(provider)
	return nil
}

func (r *providerServiceTestRepo) BulkUpdate(_ context.Context, ids []int64, updates providercore.ProviderBulkUpdate) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bulkUpdates = append(r.bulkUpdates, updates)
	return int64(len(ids)), nil
}

func (r *providerServiceTestRepo) GetByID(_ context.Context, id int64) (*providercore.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	provider := r.providers[id]
	if provider == nil {
		return nil, providercore.ErrProviderNotFound
	}
	clone := *provider
	clone.Credentials = providercore.CRSMergeMap(nil, provider.Credentials)
	clone.Extra = providercore.CRSMergeMap(nil, provider.Extra)
	clone.LoadLocation = time.LoadLocation
	return &clone, nil
}

func (r *providerServiceTestRepo) GetByIDs(_ context.Context, ids []int64) ([]*providercore.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]*providercore.Record, 0, len(ids))
	for _, id := range ids {
		if provider := r.providers[id]; provider != nil {
			result = append(result, provider)
		}
	}
	return result, nil
}

func (r *providerServiceTestRepo) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	provider := r.providers[id]
	if provider == nil {
		return providercore.ErrProviderNotFound
	}
	if provider.Extra == nil {
		provider.Extra = make(map[string]any)
	}
	for key, value := range updates {
		provider.Extra[key] = value
	}
	if r.updates == nil {
		r.updates = make(map[int64][]map[string]any)
	}
	r.updates[id] = append(r.updates[id], updates)
	return nil
}

func (r *providerServiceTestRepo) FindByExtraField(_ context.Context, key string, value any) ([]providercore.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]providercore.Record, 0)
	for _, provider := range r.providers {
		if provider.Extra != nil && provider.Extra[key] == value {
			result = append(result, *provider)
		}
	}
	return result, nil
}
