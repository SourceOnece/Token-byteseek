package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// provideBackgroundRefresh 直接绑定唯一刷新运行实例，构造不启动维护任务。
func provideBackgroundRefresh(store *postgres.ProviderStore, refresh *provider.OAuthRefreshAPI, cfg *config.Config, registrations providerRefreshRegistrations, post *provider.RefreshPostActions, observer provider.RefreshFailureObserver) *provider.BackgroundRefreshService {
	v := cfg.TokenRefresh
	tuning := &provider.RefreshTuning{
		Enabled: v.Enabled, CheckIntervalMinutes: v.CheckIntervalMinutes,
		RefreshBeforeExpiryHours: v.RefreshBeforeExpiryHours, MaxRetries: v.MaxRetries,
		RetryBackoffSeconds: v.RetryBackoffSeconds, CandidatePageSize: v.CandidatePageSize,
		ProviderConcurrency: v.ProviderConcurrency, ProviderQPS: v.ProviderQPS,
		ProviderFailureThreshold: v.ProviderFailureThreshold,
		AttemptTimeoutSeconds:    v.AttemptTimeoutSeconds, CycleTimeoutSeconds: v.CycleTimeoutSeconds,
	}
	lease, configured := refresh.LockLease()
	prepare := func(value *provider.Record) func(time.Time, string) {
		return provider.PrepareRefreshFailureNotice(observer, value)
	}
	attempts := provider.RefreshAttempts{
		API: refresh, Tuning: tuning, Policy: provider.DefaultBackgroundRefreshPolicy(),
		AttemptTimeout: tuning.AttemptTimeout(0, lease, configured),
		Now:            time.Now, Info: slog.Info, Warn: slog.Warn, Error: slog.Error,
		NonRetryable:         provideradapter.IsNonRetryableRefreshError,
		SharedProviderError:  provideradapter.IsSharedProviderRefreshError,
		AmbiguousEntitlement: provideradapter.IsAmbiguousGrokEntitlementRefreshError,
		FailureWriter:        store, GrokMutation: store, Invalidate: post.Invalidate,
		PrepareFailure: prepare, ClearRefreshRequest: post.ClearRefreshRequest,
		PostActions: post.Run, SyncCleanup: post.SyncWithCleanup,
		Persist: func(ctx context.Context, value *provider.Record, credentials map[string]any) error {
			_, err := provider.PersistCredentials(ctx, store, value, credentials, slog.Warn)
			return err
		},
	}
	return provider.NewBackgroundRefreshService(provider.BackgroundRefreshOptions{
		Tuning: tuning, Pager: store, Registrations: registrations, Attempts: attempts,
		Debug: slog.Debug, Info: slog.Info, Warn: slog.Warn, Error: slog.Error,
		Reconciliation: provider.GrokReconciliationOptions{
			Reader: store, ConditionalError: store, Now: time.Now,
			Skew: provider.GrokTokenRefreshSkew, PrepareFailure: prepare, Invalidate: post.Invalidate,
		},
	})
}
