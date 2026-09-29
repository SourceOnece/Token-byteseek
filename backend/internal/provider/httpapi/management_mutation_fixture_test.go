package httpapi

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	"github.com/TokenFlux/TokenRouter/internal/billing"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
)

func (s *managementMutationFixture) GetProvider(ctx context.Context, id int64) (*providercore.Record, error) {
	for i := range s.providers {
		if s.providers[i].ID == id {
			provider := s.providers[i]
			return &provider, nil
		}
	}
	provider := providercore.Record{ID: id, Name: "provider", Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive}
	return &provider, nil
}

func (s *managementMutationFixture) GetProvidersByIDs(ctx context.Context, ids []int64) ([]*providercore.Record, error) {
	out := make([]*providercore.Record, 0, len(ids))
	for _, id := range ids {
		found := false
		for i := range s.providers {
			if s.providers[i].ID == id {
				provider := s.providers[i]
				out = append(out, &provider)
				found = true
				break
			}
		}
		if found {
			continue
		}
		provider := providercore.Record{ID: id, Name: "provider", Status: billing.StatusActive}
		out = append(out, &provider)
	}
	return out, nil
}

func (s *managementMutationFixture) UpdateProvider(ctx context.Context, id int64, input *providercore.UpdateProviderInput) (*providercore.Record, error) {
	// 夹具模拟锁内身份条件；无条件管理请求继续保留原行为。
	if input.ExpectedCredentials != nil {
		for i := range s.providers {
			if s.providers[i].ID == id && !providercore.MatchesCredentialVersion(&s.providers[i], *input.ExpectedCredentials) {
				return nil, providercore.ErrRefreshProviderStateChanged
			}
		}
	}

	if s.updateProviderErr != nil {
		return nil, s.updateProviderErr
	}
	s.updateProviderInput = input
	for i := range s.providers {
		if s.providers[i].ID == id {
			if input.Credentials != nil {
				if input.PatchCredentials {
					s.providers[i].Credentials = providercore.MergeCredentials(s.providers[i].Credentials, input.Credentials)
				} else {
					s.providers[i].Credentials = input.Credentials
				}
			}
			provider := s.providers[i]
			return &provider, nil
		}
	}
	provider := providercore.Record{ID: id, Name: input.Name, Platform: capability.PlatformAnthropic, Type: input.Type, Status: billing.StatusActive, Credentials: input.Credentials}
	return &provider, nil
}

func (s *managementMutationFixture) UpdateProviderExtra(ctx context.Context, id int64, updates map[string]any) error {
	s.updateExtraCalls = append(s.updateExtraCalls, updates)
	return nil
}

func (s *managementMutationFixture) ClearProviderError(ctx context.Context, id int64) (*providercore.Record, error) {
	s.clearProviderErrorIDs = append(s.clearProviderErrorIDs, id)
	provider := providercore.Record{ID: id, Name: "provider", Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive}
	return &provider, nil
}

func (s *managementMutationFixture) BulkUpdateProviders(ctx context.Context, input *providercore.BulkUpdateProvidersInput) (*providercore.BulkUpdateProvidersResult, error) {
	s.lastBulkUpdateProviderInput = input
	if s.bulkUpdateProviderErr != nil {
		return nil, s.bulkUpdateProviderErr
	}
	return &providercore.BulkUpdateProvidersResult{Success: len(input.ProviderIDs), Failed: 0, SuccessIDs: input.ProviderIDs}, nil
}

func (s *managementMutationFixture) EnsureOpenAIPrivacy(ctx context.Context, provider *providercore.Record) string {
	return ""
}

func (s *managementMutationFixture) EnsureAntigravityPrivacy(ctx context.Context, provider *providercore.Record) string {
	return ""
}

// managementMutationFixture 记录配置、凭据与失效输入，复用原独立存储替身语义。
type managementMutationFixture struct {
	managementCreateFixture
	providers                   []providercore.Record
	updateProviderInput         *providercore.UpdateProviderInput
	updateProviderErr           error
	updateExtraCalls            []map[string]any
	clearProviderErrorIDs       []int64
	lastBulkUpdateProviderInput *providercore.BulkUpdateProvidersInput
	bulkUpdateProviderErr       error
}

func newManagementMutationFixture() *managementMutationFixture { return &managementMutationFixture{} }

// newMutationHandler 直接组合提供商用例和展示层，不经过旧管理服务。
func newMutationHandler(source *managementMutationFixture, invalidator providercore.TokenCacheInvalidator) *ManagementHandler {
	options := providercore.ManagedRefreshOptions{Store: source, Privacy: source}
	if invalidator != nil {
		options.Invalidate = invalidator.InvalidateToken
	}
	managed := providercore.NewManagedRefreshService(options)
	presenter := NewRuntimePresenter(providercore.NewRuntimeStatusReader(providercore.RuntimeStatusOptions{}), source, nil)
	batch := providercore.NewManagementBatch(source, managed, providercore.ManagementCreationOptions{Privacy: source})
	return NewManagementHandler(source, ManagementOptions{Managed: managed, Presenter: presenter, RuntimePresenter: presenter, Privacy: source, Batch: batch})
}
