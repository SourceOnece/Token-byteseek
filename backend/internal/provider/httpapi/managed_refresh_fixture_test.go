package httpapi

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// newManagedRefreshFixture 保留原独立构造的单次协调，只注入当前测试的平台交换。
func newManagedRefreshFixture(source interface {
	provider.ManagedCredentialStore
	provider.ManagedCredentialPrivacy
}, exchange *provider.ManualCredentialExchange,
) *provider.ManagedRefreshService {
	return provider.NewManagedRefreshService(provider.ManagedRefreshOptions{
		Store: source, Privacy: source, CacheKey: provideradapter.ManagedRefreshCacheKey,
		Coordinate: func(ctx context.Context, v *provider.Record, _ string, apply func(context.Context, *provider.Record) (*provider.Record, string, error)) (*provider.Record, string, error) {
			return apply(ctx, v)
		},
		Exchange: func(ctx context.Context, v *provider.Record) (provider.ManagedRefreshObservation, error) {
			credentials, missing, err := exchange.Refresh(ctx, v)
			return provider.ManagedRefreshObservation{Credentials: credentials, ProjectIDMissing: missing}, err
		},
	})
}
