package provider

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

type grokReconcileRepo struct {
	mu                      sync.Mutex
	providers               []providercore.Record
	requests                []providercore.OAuthRefreshPageOptions
	setErrorIDs             []int64
	updatedCredIDs          []int64
	setErrorMessage         []string
	getByIDOverrides        map[int64]providercore.Record
	pageOverride            *providercore.OAuthRefreshCandidatePage
	reauthorizeOnCAS        bool
	reauthorizeOnRefreshCAS bool
	conditionalCalls        int
}

func (r *grokReconcileRepo) GetByID(_ context.Context, id int64) (*providercore.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if override, ok := r.getByIDOverrides[id]; ok {
		provider := override
		return &provider, nil
	}
	for i := range r.providers {
		if r.providers[i].ID == id {
			provider := r.providers[i]
			return &provider, nil
		}
	}
	return nil, providercore.ErrProviderNotFound
}

func (r *grokReconcileRepo) ListOAuthRefreshCandidatePage(_ context.Context, options providercore.OAuthRefreshPageOptions) (*providercore.OAuthRefreshCandidatePage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, options)
	if r.pageOverride != nil {
		page := *r.pageOverride
		page.Providers = append([]providercore.Record(nil), r.pageOverride.Providers...)
		return &page, nil
	}
	providers := append([]providercore.Record(nil), r.providers...)
	sort.Slice(providers, func(i, j int) bool { return providers[i].ID < providers[j].ID })
	page := make([]providercore.Record, 0, options.Limit)
	for _, provider := range providers {
		if provider.ID <= options.AfterID {
			continue
		}
		platformAllowed := false
		for _, platform := range options.Platforms {
			if provider.Platform == platform {
				platformAllowed = true
				break
			}
		}
		if !platformAllowed || options.ActiveOnly && provider.Status != providercore.StatusActive {
			continue
		}
		if options.IncludeSetupToken {
			if provider.Type != capability.ProviderTypeOAuth && provider.Type != capability.ProviderTypeSetupToken {
				continue
			}
		} else if provider.Type != capability.ProviderTypeOAuth {
			continue
		}
		if options.RequireRefreshToken && strings.TrimSpace(provider.GetGrokRefreshToken()) == "" {
			continue
		}
		page = append(page, provider)
		if len(page) == options.Limit {
			break
		}
	}
	result := &providercore.OAuthRefreshCandidatePage{Providers: page, HasMore: len(page) == options.Limit}
	if len(page) > 0 {
		result.NextAfterID = page[len(page)-1].ID
	}
	return result, nil
}

func (r *grokReconcileRepo) UpdateCredentials(_ context.Context, id int64, credentials map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updatedCredIDs = append(r.updatedCredIDs, id)
	for i := range r.providers {
		if r.providers[i].ID == id {
			r.providers[i].Credentials = providercore.MergeCredentials(r.providers[i].Credentials, credentials)
		}
	}
	return nil
}

func (r *grokReconcileRepo) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.providers {
		if r.providers[i].ID != id {
			continue
		}
		if r.providers[i].Extra == nil {
			r.providers[i].Extra = make(map[string]any)
		}
		for key, value := range updates {
			r.providers[i].Extra[key] = value
		}
		break
	}
	return nil
}

func (r *grokReconcileRepo) SetError(_ context.Context, id int64, message string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.setErrorIDs = append(r.setErrorIDs, id)
	r.setErrorMessage = append(r.setErrorMessage, message)
	for i := range r.providers {
		if r.providers[i].ID == id {
			r.providers[i].Status = providercore.StatusError
			r.providers[i].Schedulable = false
			r.providers[i].ErrorMessage = message
		}
	}
	return nil
}

func (r *grokReconcileRepo) SetGrokOAuthErrorIfCredentialsUnchanged(
	_ context.Context,
	id int64,
	expectedCredentials map[string]any,
	message string,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conditionalCalls++
	for i := range r.providers {
		provider := &r.providers[i]
		if provider.ID != id {
			continue
		}
		if r.reauthorizeOnCAS {
			r.reauthorizeOnCAS = false
			provider.Credentials = map[string]any{
				"access_token":   "fresh-access",
				"refresh_token":  "fresh-refresh",
				"expires_at":     time.Now().UTC().Add(4 * time.Hour).Format(time.RFC3339),
				"_token_version": int64(2),
			}
		}
		if provider.Platform != capability.PlatformGrok || provider.Type != capability.ProviderTypeOAuth || provider.Status != providercore.StatusActive ||
			strings.TrimSpace(provider.GetGrokRefreshToken()) != "" || !reflect.DeepEqual(provider.Credentials, expectedCredentials) {
			return false, nil
		}
		r.setErrorIDs = append(r.setErrorIDs, id)
		r.setErrorMessage = append(r.setErrorMessage, message)
		provider.Status = providercore.StatusError
		provider.Schedulable = false
		provider.ErrorMessage = message
		return true, nil
	}
	return false, nil
}

func (r *grokReconcileRepo) SetGrokOAuthRefreshErrorIfCredentialsUnchanged(
	_ context.Context,
	id int64,
	expectedCredentials map[string]any,
	expectedProxyID *int64,
	message string,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.providers {
		provider := &r.providers[i]
		if provider.ID != id {
			continue
		}
		if r.reauthorizeOnRefreshCAS {
			r.reauthorizeOnRefreshCAS = false
			provider.Credentials = map[string]any{
				"access_token":   "fresh-access",
				"refresh_token":  "fresh-refresh",
				"expires_at":     time.Now().UTC().Add(4 * time.Hour).Format(time.RFC3339),
				"_token_version": int64(3),
			}
		}
		if provider.Platform != capability.PlatformGrok || provider.Type != capability.ProviderTypeOAuth || provider.Status != providercore.StatusActive ||
			!reflect.DeepEqual(provider.ProxyID, expectedProxyID) ||
			!reflect.DeepEqual(provider.Credentials, expectedCredentials) {
			return false, nil
		}
		r.setErrorIDs = append(r.setErrorIDs, id)
		r.setErrorMessage = append(r.setErrorMessage, message)
		provider.Status = providercore.StatusError
		provider.Schedulable = false
		provider.ErrorMessage = message
		return true, nil
	}
	return false, nil
}

func (r *grokReconcileRepo) SetGrokOAuthRefreshTempUnschedulableIfCredentialsUnchanged(
	_ context.Context,
	id int64,
	expectedCredentials map[string]any,
	expectedProxyID *int64,
	until time.Time,
	reason string,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.providers {
		provider := &r.providers[i]
		if provider.ID != id {
			continue
		}
		if provider.Platform != capability.PlatformGrok || provider.Type != capability.ProviderTypeOAuth || provider.Status != providercore.StatusActive ||
			!reflect.DeepEqual(provider.ProxyID, expectedProxyID) ||
			!reflect.DeepEqual(provider.Credentials, expectedCredentials) {
			return false, nil
		}
		provider.TempUnschedulableUntil = &until
		provider.TempUnschedulableReason = reason
		return true, nil
	}
	return false, nil
}

func (r *grokReconcileRepo) snapshot() ([]providercore.OAuthRefreshPageOptions, []int64, []int64, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]providercore.OAuthRefreshPageOptions(nil), r.requests...), append([]int64(nil), r.setErrorIDs...), append([]int64(nil), r.updatedCredIDs...), append([]string(nil), r.setErrorMessage...)
}

type reconcileInvalidator struct {
	mu  sync.Mutex
	ids []int64
	err error
}

type reconcileRuntimeBlocker struct {
	mu      sync.Mutex
	blocked []int64
	cleared []int64
}

func (b *reconcileRuntimeBlocker) BlockProviderScheduling(provider *providercore.Record, _ time.Time, _ string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if provider != nil {
		b.blocked = append(b.blocked, provider.ID)
	}
}

func (b *reconcileRuntimeBlocker) ClearProviderSchedulingBlock(providerID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cleared = append(b.cleared, providerID)
}

func (b *reconcileRuntimeBlocker) snapshot() (blocked, cleared []int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]int64(nil), b.blocked...), append([]int64(nil), b.cleared...)
}

func (i *reconcileInvalidator) InvalidateToken(_ context.Context, provider *providercore.Record) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.ids = append(i.ids, provider.ID)
	return i.err
}

func (i *reconcileInvalidator) count() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return len(i.ids)
}

func newGrokReconcileService(repo *grokReconcileRepo, refresher *poolHealthRefresher, invalidator providercore.TokenCacheInvalidator, observers ...providercore.RefreshFailureObserver) *providercore.BackgroundRefreshService {
	tuning := &providercore.RefreshTuning{
		MaxRetries:               1,
		CandidatePageSize:        50,
		ProviderConcurrency:      2,
		ProviderQPS:              100,
		ProviderFailureThreshold: 3,
		AttemptTimeoutSeconds:    1,
	}
	var observer providercore.RefreshFailureObserver
	if len(observers) > 0 {
		observer = observers[0]
	}
	prepare := func(value *providercore.Record) func(time.Time, string) {
		return providercore.PrepareRefreshFailureNotice(observer, value)
	}
	attempts := backgroundAttemptOptions(repo, tuning)
	attempts.PrepareFailure = prepare
	var invalidate func(context.Context, *providercore.Record) error
	if invalidator != nil {
		invalidate = invalidator.InvalidateToken
	}
	attempts.Invalidate = invalidate
	post := &providercore.RefreshPostActions{
		Now: time.Now, Info: func(string, ...any) {}, Warn: func(string, ...any) {}, Debug: func(string, ...any) {}, Invalidate: invalidate, ClearBlock: func(int64) {}, NeedsReauth: providercore.GrokNeedsReauth,
		ClearReauth: func(ctx context.Context, value *providercore.Record) {
			providercore.ClearGrokNeedsReauth(ctx, repo, value.ID)
		},
	}
	attempts.PostActions, attempts.SyncCleanup = post.Run, post.SyncWithCleanup
	return providercore.NewBackgroundRefreshService(providercore.BackgroundRefreshOptions{
		Tuning: tuning, Pager: repo, Attempts: attempts,
		Registrations:  []providercore.RefreshRegistration{{Platform: capability.PlatformGrok, Refresher: refresher, Executor: refresher}},
		Reconciliation: providercore.GrokReconciliationOptions{Reader: repo, ConditionalError: repo, PrepareFailure: prepare, Invalidate: invalidate, Now: time.Now, Skew: providercore.GrokTokenRefreshSkew},
	})
}

func grokReconcileFixtures() []providercore.Record {
	now := time.Now().UTC()
	return []providercore.Record{
		{
			ID:          1,
			Platform:    capability.PlatformGrok,
			Type:        capability.ProviderTypeOAuth,
			Status:      providercore.StatusActive,
			Schedulable: true,
			Credentials: map[string]any{"access_token": "access-secret"},
		},
		{
			ID:          2,
			Platform:    capability.PlatformGrok,
			Type:        capability.ProviderTypeOAuth,
			Status:      providercore.StatusActive,
			Schedulable: true,
			Credentials: map[string]any{"refresh_token": "refresh-secret", "expires_at": now.Add(10 * time.Minute).Format(time.RFC3339)},
		},
		{
			ID:          3,
			Platform:    capability.PlatformGrok,
			Type:        capability.ProviderTypeOAuth,
			Status:      providercore.StatusActive,
			Schedulable: true,
			Credentials: map[string]any{"access_token": "access-secret", "refresh_token": "refresh-secret", "expires_at": now.Add(30 * time.Minute).Format(time.RFC3339)},
		},
		{
			ID:          4,
			Platform:    capability.PlatformGrok,
			Type:        capability.ProviderTypeOAuth,
			Status:      providercore.StatusActive,
			Schedulable: true,
			Credentials: map[string]any{"access_token": "access-secret", "refresh_token": "refresh-secret", "expires_at": now.Add(4 * time.Hour).Format(time.RFC3339)},
		},
		{
			ID:          5,
			Platform:    capability.PlatformGrok,
			Type:        capability.ProviderTypeAPIKey,
			Status:      providercore.StatusActive,
			Schedulable: true,
			Credentials: map[string]any{"api_key": "api-key-secret"},
		},
	}
}

func TestTokenRefreshService_ReconcileGrokOAuthDefaultsToDryRunAndSanitizedPlan(t *testing.T) {
	repo := &grokReconcileRepo{providers: grokReconcileFixtures()}
	refresher := &poolHealthRefresher{}
	svc := newGrokReconcileService(repo, refresher, nil)

	result, err := svc.ReconcileGrokOAuth(context.Background(), providercore.GrokOAuthReconcileInput{})

	require.NoError(t, err)
	require.True(t, result.DryRun)
	require.Equal(t, 4, result.Scanned, "Grok API-key rows must not enter the OAuth reconciliation page")
	require.Equal(t, 3, result.Actionable)
	require.Equal(t, 1, result.WouldBlock)
	require.Equal(t, 2, result.WouldRefresh)
	require.Zero(t, result.Blocked)
	require.Zero(t, result.Refreshed)
	require.Zero(t, refresher.calls.Load())
	_, setErrorIDs, updatedIDs, _ := repo.snapshot()
	require.Empty(t, setErrorIDs)
	require.Empty(t, updatedIDs)

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	text := string(payload)
	require.NotContains(t, text, "access-secret")
	require.NotContains(t, text, "refresh-secret")
	require.NotContains(t, text, "api-key-secret")
	require.NotContains(t, text, `"credentials":`)
}

func TestGrokTokenRefresher_NeedsRefreshWhenAccessTokenMissingDespiteFarFutureExpiry(t *testing.T) {
	refresher := providercore.NewGrokTokenRefresher(nil)
	provider := grokPoolProvider(99)
	delete(provider.Credentials, "access_token")
	provider.Credentials["expires_at"] = time.Now().UTC().Add(12 * time.Hour).Format(time.RFC3339)

	require.True(t, refresher.NeedsRefresh(&provider, time.Hour))
}

func TestTokenRefreshService_ReconcileGrokOAuthApplyIsIdempotent(t *testing.T) {
	repo := &grokReconcileRepo{providers: grokReconcileFixtures()}
	invalidator := &reconcileInvalidator{}
	refresher := &poolHealthRefresher{newCredentials: map[string]any{
		"access_token":  "rotated-access-secret",
		"refresh_token": "rotated-refresh-secret",
		"expires_at":    time.Now().UTC().Add(4 * time.Hour).Format(time.RFC3339),
	}}
	svc := newGrokReconcileService(repo, refresher, invalidator)

	first, err := svc.ReconcileGrokOAuth(context.Background(), providercore.GrokOAuthReconcileInput{Apply: true, Limit: 50})
	require.NoError(t, err)
	require.False(t, first.DryRun)
	require.Equal(t, 1, first.Blocked)
	require.Equal(t, 2, first.Refreshed)
	require.Zero(t, first.Failed)
	requests, setErrorIDs, updatedIDs, messages := repo.snapshot()
	require.Equal(t, []int64{1}, setErrorIDs)
	sort.Slice(updatedIDs, func(i, j int) bool { return updatedIDs[i] < updatedIDs[j] })
	require.Equal(t, []int64{2, 3}, updatedIDs)
	require.Len(t, messages, 1)
	require.NotContains(t, messages[0], "secret")
	require.False(t, requests[0].RequireRefreshToken, "structurally invalid rows must remain discoverable")
	require.False(t, requests[0].IncludeSetupToken)
	require.Equal(t, 3, invalidator.count(), "block and refresh actions must invalidate token cache state")

	second, err := svc.ReconcileGrokOAuth(context.Background(), providercore.GrokOAuthReconcileInput{Apply: true, Limit: 50})
	require.NoError(t, err)
	require.Zero(t, second.Actionable)
	require.Equal(t, int64(2), refresher.calls.Load(), "already refreshed rows must not be refreshed again")
	_, setErrorIDs, updatedIDs, _ = repo.snapshot()
	require.Equal(t, []int64{1}, setErrorIDs, "already blocked invalid rows must not transition twice")
	require.Len(t, updatedIDs, 2)
}

func TestTokenRefreshService_ReconcileGrokOAuthCursorResumesWithoutDuplicates(t *testing.T) {
	fixtures := grokReconcileFixtures()[:3]
	repo := &grokReconcileRepo{providers: fixtures}
	svc := newGrokReconcileService(repo, &poolHealthRefresher{}, nil)

	first, err := svc.ReconcileGrokOAuth(context.Background(), providercore.GrokOAuthReconcileInput{Limit: 2})
	require.NoError(t, err)
	require.True(t, first.HasMore)
	require.Equal(t, int64(2), first.NextAfterID)
	require.Len(t, first.Items, 2)

	second, err := svc.ReconcileGrokOAuth(context.Background(), providercore.GrokOAuthReconcileInput{AfterID: first.NextAfterID, Limit: 2})
	require.NoError(t, err)
	require.False(t, second.HasMore)
	require.Zero(t, second.NextAfterID)
	require.Len(t, second.Items, 1)
	require.NotEqual(t, first.Items[0].ProviderID, second.Items[0].ProviderID)
	require.NotEqual(t, first.Items[1].ProviderID, second.Items[0].ProviderID)
}

func TestTokenRefreshService_ReconcileGrokOAuthCursorUsesRawPageAfterHydrationGap(t *testing.T) {
	provider := grokReconcileFixtures()[0]
	repo := &grokReconcileRepo{pageOverride: &providercore.OAuthRefreshCandidatePage{
		Providers:   []providercore.Record{provider},
		NextAfterID: provider.ID + 1,
		HasMore:     true,
	}}
	svc := newGrokReconcileService(repo, &poolHealthRefresher{}, nil)

	result, err := svc.ReconcileGrokOAuth(context.Background(), providercore.GrokOAuthReconcileInput{Limit: 2})

	require.NoError(t, err)
	require.True(t, result.HasMore)
	require.Equal(t, provider.ID+1, result.NextAfterID,
		"cursor must advance past a raw selected ID that disappeared during hydration")
}

func TestTokenRefreshService_ReconcileGrokOAuthRejectsConflictingApplyMode(t *testing.T) {
	svc := newGrokReconcileService(&grokReconcileRepo{}, &poolHealthRefresher{}, nil)

	_, err := svc.ReconcileGrokOAuth(context.Background(), providercore.GrokOAuthReconcileInput{Apply: true, DryRun: true})

	require.ErrorIs(t, err, providercore.ErrGrokOAuthReconcileMode)
}

func TestTokenRefreshService_ReconcileGrokOAuthSkipsStaleBlockAfterConcurrentReauthorization(t *testing.T) {
	stale := grokReconcileFixtures()[0]
	latest := stale
	latest.Credentials = map[string]any{
		"access_token":  "fresh-access",
		"refresh_token": "fresh-refresh",
		"expires_at":    time.Now().UTC().Add(4 * time.Hour).Format(time.RFC3339),
	}
	repo := &grokReconcileRepo{
		providers:        []providercore.Record{stale},
		getByIDOverrides: map[int64]providercore.Record{stale.ID: latest},
	}
	svc := newGrokReconcileService(repo, &poolHealthRefresher{}, &reconcileInvalidator{})

	result, err := svc.ReconcileGrokOAuth(context.Background(), providercore.GrokOAuthReconcileInput{Apply: true, Limit: 50})

	require.NoError(t, err)
	require.Zero(t, result.Blocked)
	require.Equal(t, 1, result.Skipped)
	require.Equal(t, providercore.GrokOAuthReconcileOutcomeSkipped, result.Items[0].Outcome)
	_, setErrorIDs, _, _ := repo.snapshot()
	require.Empty(t, setErrorIDs, "a concurrently reauthorized provider must not be disabled from stale page state")
}

func TestTokenRefreshService_ReconcileGrokOAuthDoesNotRuntimeBlockWhenReauthorizationWinsConditionalMutation(t *testing.T) {
	provider := grokReconcileFixtures()[0]
	provider.Credentials["_token_version"] = int64(1)
	repo := &grokReconcileRepo{
		providers:        []providercore.Record{provider},
		reauthorizeOnCAS: true,
	}
	invalidator := &reconcileInvalidator{}
	blocker := &reconcileRuntimeBlocker{}
	svc := newGrokReconcileService(repo, &poolHealthRefresher{}, invalidator, blocker)

	result, err := svc.ReconcileGrokOAuth(context.Background(), providercore.GrokOAuthReconcileInput{Apply: true, Limit: 50})

	require.NoError(t, err)
	require.Zero(t, result.Blocked)
	require.Equal(t, 1, result.Skipped)
	require.Equal(t, providercore.GrokOAuthReconcileOutcomeSkipped, result.Items[0].Outcome)
	require.Zero(t, invalidator.count(), "a lost compare-and-set race must not invalidate fresh credentials")
	_, setErrorIDs, _, _ := repo.snapshot()
	require.Empty(t, setErrorIDs)
	require.Equal(t, 1, repo.conditionalCalls)
	blocked, cleared := blocker.snapshot()
	require.Empty(t, blocked, "a lost compare-and-set race must never install a runtime block")
	require.Empty(t, cleared, "reconciliation must not clear a block it does not own")

	latest, getErr := repo.GetByID(context.Background(), provider.ID)
	require.NoError(t, getErr)
	require.Equal(t, providercore.StatusActive, latest.Status)
	require.True(t, latest.Schedulable)
	require.Equal(t, "fresh-refresh", latest.GetGrokRefreshToken())
}

func TestTokenRefreshService_ReconcileGrokOAuthReportsPermanentRefreshMutationAsBlocked(t *testing.T) {
	provider := grokReconcileFixtures()[2]
	repo := &grokReconcileRepo{providers: []providercore.Record{provider}}
	refresher := &poolHealthRefresher{err: errors.New(`GROK_OAUTH_ENTITLEMENT_DENIED: subscription required`)}
	svc := newGrokReconcileService(repo, refresher, &reconcileInvalidator{})

	result, err := svc.ReconcileGrokOAuth(context.Background(), providercore.GrokOAuthReconcileInput{Apply: true, Limit: 50})

	require.NoError(t, err)
	require.Equal(t, 1, result.Blocked)
	require.Zero(t, result.Failed)
	require.Zero(t, result.Partial)
	require.Equal(t, providercore.GrokOAuthReconcileActionBlock, result.Items[0].Action)
	require.Equal(t, providercore.GrokOAuthReconcileReasonCredentialRejected, result.Items[0].Reason)
	require.Equal(t, providercore.GrokOAuthReconcileOutcomeApplied, result.Items[0].Outcome)
	_, setErrorIDs, _, _ := repo.snapshot()
	require.Equal(t, []int64{provider.ID}, setErrorIDs)
}

func TestTokenRefreshService_ReconcileGrokOAuthReportsConcurrentRefreshReauthorizationAsSkipped(t *testing.T) {
	provider := grokReconcileFixtures()[2]
	repo := &grokReconcileRepo{
		providers:               []providercore.Record{provider},
		reauthorizeOnRefreshCAS: true,
	}
	invalidator := &reconcileInvalidator{}
	refresher := &poolHealthRefresher{err: errors.New("invalid_grant: revoked")}
	svc := newGrokReconcileService(repo, refresher, invalidator)

	result, err := svc.ReconcileGrokOAuth(context.Background(), providercore.GrokOAuthReconcileInput{Apply: true, Limit: 50})

	require.NoError(t, err)
	require.Equal(t, 1, result.Skipped)
	require.Zero(t, result.Failed)
	require.Zero(t, result.Blocked)
	require.Equal(t, providercore.GrokOAuthReconcileOutcomeSkipped, result.Items[0].Outcome)
	require.Zero(t, invalidator.count())
	_, setErrorIDs, _, _ := repo.snapshot()
	require.Empty(t, setErrorIDs)
	latest, getErr := repo.GetByID(context.Background(), provider.ID)
	require.NoError(t, getErr)
	require.Equal(t, providercore.StatusActive, latest.Status)
	require.Equal(t, "fresh-refresh", latest.GetGrokRefreshToken())
}

func TestTokenRefreshService_ReconcileGrokOAuthReportsInvalidationFailureAsPartial(t *testing.T) {
	provider := grokReconcileFixtures()[0]
	repo := &grokReconcileRepo{providers: []providercore.Record{provider}}
	svc := newGrokReconcileService(repo, &poolHealthRefresher{}, &reconcileInvalidator{err: errors.New("cache unavailable")})

	result, err := svc.ReconcileGrokOAuth(context.Background(), providercore.GrokOAuthReconcileInput{Apply: true, Limit: 50})

	require.NoError(t, err)
	require.Equal(t, 1, result.Blocked)
	require.Equal(t, 1, result.Partial)
	require.Zero(t, result.Failed)
	require.Equal(t, providercore.GrokOAuthReconcileOutcomePartial, result.Items[0].Outcome)
}
