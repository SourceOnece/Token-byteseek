package app

import (
	"context"
	"log"
	"log/slog"

	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// provideManagedRefresh 绑定原生提供商用例与唯一刷新协调器，构造不执行交换。
func provideManagedRefresh(admin *provider.Admin, privacy *provider.PrivacyService, coordinator *provider.OAuthRefreshAPI, transport httpclient.UpstreamTransport, profiles *egressprovider.TLSProfiles, claude *provider.ClaudeAuthorization, openai *provider.OpenAIAuthorization, gemini *provider.GeminiAuthorization, ag *provider.AntigravityAuthorization, grok provider.GrokRefreshTokenService, invalidator provider.TokenCacheInvalidator) *provider.ManagedRefreshService {
	qoder := provideradapter.NewQoderTokenRefresher(provideradapter.QoderRefreshOptions{Transport: transport, Profiles: profiles})
	source := &provider.ManualCredentialExchange{Claude: claude, OpenAI: openai, Gemini: gemini, Antigravity: ag, Grok: grok, Qoder: qoder.Refresh}
	exchange := func(ctx context.Context, value *provider.Record) (provider.ManagedRefreshObservation, error) {
		credentials, missing, err := source.Refresh(ctx, value)
		return provider.ManagedRefreshObservation{Credentials: credentials, ProjectIDMissing: missing}, err
	}
	var invalidate func(context.Context, *provider.Record) error
	if invalidator != nil {
		invalidate = func(ctx context.Context, value *provider.Record) error {
			return invalidator.InvalidateToken(ctx, value)
		}
	}
	return provider.NewManagedRefreshService(provider.ManagedRefreshOptions{Store: admin, Privacy: privacy, Coordinate: coordinator.WithManagedRefresh, CacheKey: provideradapter.ManagedRefreshCacheKey, Exchange: exchange, Invalidate: invalidate, Log: log.Printf, Warn: slog.Warn, Error: slog.Error})
}
