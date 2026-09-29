package provider

import (
	"context"
	"log/slog"
	"reflect"
	"time"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
)

// backgroundAttemptOptions 只绑定替身端口，实际周期、尝试和成功清理由原生组件执行。
func backgroundAttemptOptions(repo providercore.CredentialUpdateStore, tuning *providercore.RefreshTuning) providercore.RefreshAttempts {
	post := &providercore.RefreshPostActions{
		Now: time.Now, Info: slog.Info, Warn: slog.Warn, Debug: slog.Debug,
		ClearBlock: func(int64) {}, NeedsReauth: providercore.GrokNeedsReauth,
		ClearReauth: func(context.Context, *providercore.Record) {},
	}
	failure, _ := repo.(providercore.RefreshFailureWriter)
	grok, _ := repo.(providercore.GrokRefreshMutationWriter)
	return providercore.RefreshAttempts{
		Tuning: tuning, Policy: providercore.DefaultBackgroundRefreshPolicy(),
		AttemptTimeout: tuning.AttemptTimeout(0, 0, false), Now: time.Now,
		Info: slog.Info, Warn: slog.Warn, Error: slog.Error,
		NonRetryable: IsNonRetryableRefreshError, SharedProviderError: IsSharedProviderRefreshError,
		AmbiguousEntitlement: IsAmbiguousGrokEntitlementRefreshError,
		FailureWriter:        failure, GrokMutation: grok,
		PrepareFailure:      func(*providercore.Record) func(time.Time, string) { return func(time.Time, string) {} },
		ClearRefreshRequest: post.ClearRefreshRequest, PostActions: post.Run, SyncCleanup: post.SyncWithCleanup,
		Persist: func(ctx context.Context, value *providercore.Record, credentials map[string]any) error {
			_, err := providercore.PersistCredentials(ctx, repo, value, credentials, slog.Warn)
			return err
		},
	}
}

func refreshAPIForFixture(repo providercore.RefreshRepository, cache providercore.RefreshCache) *providercore.OAuthRefreshAPI {
	return providercore.NewOAuthRefreshAPI(repo, cache, providercore.RefreshOptions{Platform: providercore.ProviderRefreshPlatformPolicy()})
}

// 测试写入保留字段计数；任何意外的整行更新仍通过同一替身可观测。
func (r *poolHealthProviderRepo) Update(ctx context.Context, value *providercore.Record) error {
	return r.UpdateCredentials(ctx, value.ID, value.Credentials)
}

func (r *productionPathRateRepo) Update(ctx context.Context, value *providercore.Record) error {
	return r.UpdateCredentials(ctx, value.ID, value.Credentials)
}

func (r *grokReconcileRepo) Update(ctx context.Context, value *providercore.Record) error {
	return r.UpdateCredentials(ctx, value.ID, value.Credentials)
}

func (b *reconcileRuntimeBlocker) PrepareRefreshFailure(int64) func(providercore.RefreshFailureNotice) {
	return func(value providercore.RefreshFailureNotice) {
		b.BlockProviderScheduling(&providercore.Record{ID: value.ProviderID}, value.Until, value.Reason)
	}
}

func (r *tokenRefreshCandidateRepo) Update(ctx context.Context, value *providercore.Record) error {
	return r.UpdateCredentials(ctx, value.ID, value.Credentials)
}

func candidatePostActions(repo *tokenRefreshCandidateRepo) *providercore.RefreshPostActions {
	return &providercore.RefreshPostActions{
		Now: time.Now, Info: slog.Info, Warn: slog.Warn, Debug: slog.Debug, ClearBlock: func(int64) {}, NeedsReauth: providercore.GrokNeedsReauth,
		ClearCooldown: func(ctx context.Context, value *providercore.Record) (bool, error) {
			return repo.ClearRefreshCooldownIfUnchanged(ctx, providercore.ObserveRefreshCooldown(value))
		},
	}
}

func cooldownPostActions(repo *refreshSuccessCooldownRepo) *providercore.RefreshPostActions {
	return &providercore.RefreshPostActions{
		Now: time.Now, Info: slog.Info, Warn: slog.Warn, Debug: slog.Debug, ClearBlock: func(int64) {}, NeedsReauth: providercore.GrokNeedsReauth,
		ClearCooldown: func(ctx context.Context, value *providercore.Record) (bool, error) {
			return repo.ClearRefreshCooldownIfUnchanged(ctx, providercore.ObserveRefreshCooldown(value))
		},
	}
}

// 竞争替身比较当前行身份，nil 凭据与原刷新快照一致。
func refreshFailureMatchesFixture(value *providercore.Record, version providercore.RefreshFailureVersion) bool {
	return value != nil && reflect.DeepEqual(providercore.FailureVersion(value), version)
}

func (r *tokenRefreshCandidateRepo) ApplyOAuthRefreshFailure(ctx context.Context, version providercore.RefreshFailureVersion, failure providercore.RefreshFailure) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	matched := r.providers == nil
	for i := range r.providers {
		if r.providers[i].ID == version.ID {
			matched = refreshFailureMatchesFixture(&r.providers[i], version)
			break
		}
	}
	if !matched {
		return false, nil
	}
	if failure.Kind == providercore.RefreshFailurePermanent {
		r.setErrorCalls++
	} else {
		r.setTempUnschedCalls++
		r.lastTempUnschedReason = failure.Message
	}
	return true, nil
}
