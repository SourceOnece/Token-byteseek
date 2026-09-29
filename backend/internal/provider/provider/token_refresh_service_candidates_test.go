package provider

import (
	"context"
	"errors"
	"maps"
	"strings"
	"sync"
	"testing"
	"time"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/require"
)

type tokenRefreshCandidateRepo struct {
	mu                    sync.Mutex
	providers             []providercore.Record
	updatedCredentialIDs  []int64
	setErrorCalls         int
	setTempUnschedCalls   int
	clearTempCalls        int
	lastTempUnschedReason string
	listActiveCalls       int
}

func (r *tokenRefreshCandidateRepo) ListActive(context.Context) ([]providercore.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listActiveCalls++
	return r.providers, nil
}

func (r *tokenRefreshCandidateRepo) ListOAuthRefreshCandidatePage(_ context.Context, options providercore.OAuthRefreshPageOptions) (*providercore.OAuthRefreshCandidatePage, error) {
	candidates := make([]providercore.Record, 0, len(r.providers))
	now := time.Now()
	for _, provider := range r.providers {
		if provider.ID <= options.AfterID {
			continue
		}
		refreshToken, _ := provider.Credentials["refresh_token"].(string)
		inRetryCooldown := provider.TempUnschedulableUntil != nil &&
			provider.TempUnschedulableUntil.After(now) &&
			strings.HasPrefix(provider.TempUnschedulableReason, "token refresh retry exhausted:")
		platformAllowed := false
		for _, platform := range options.Platforms {
			if provider.Platform == platform {
				platformAllowed = true
				break
			}
		}
		typeAllowed := provider.Type == capability.ProviderTypeOAuth ||
			(options.IncludeSetupToken && provider.Type == capability.ProviderTypeSetupToken) ||
			provider.IsQoderCosy()
		if (options.ActiveOnly && provider.Status != providercore.StatusActive) ||
			!provider.Schedulable ||
			!typeAllowed ||
			!platformAllowed ||
			(options.RequireRefreshToken && strings.TrimSpace(refreshToken) == "") ||
			(options.ExcludeRetryCooldown && inRetryCooldown) {
			continue
		}
		candidates = append(candidates, provider)
		if len(candidates) == options.Limit {
			break
		}
	}
	page := &providercore.OAuthRefreshCandidatePage{Providers: candidates, HasMore: len(candidates) == options.Limit}
	if len(candidates) > 0 {
		page.NextAfterID = candidates[len(candidates)-1].ID
	}
	return page, nil
}

func (r *tokenRefreshCandidateRepo) UpdateCredentials(_ context.Context, id int64, credentials map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updatedCredentialIDs = append(r.updatedCredentialIDs, id)
	for i := range r.providers {
		if r.providers[i].ID == id {
			r.providers[i].Credentials = maps.Clone(credentials)
		}
	}
	return nil
}

func (r *tokenRefreshCandidateRepo) SetError(context.Context, int64, string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.setErrorCalls++
	return nil
}

func (r *tokenRefreshCandidateRepo) SetTempUnschedulable(_ context.Context, _ int64, _ time.Time, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.setTempUnschedCalls++
	r.lastTempUnschedReason = reason
	return nil
}

func (r *tokenRefreshCandidateRepo) ClearTempUnschedulable(context.Context, int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clearTempCalls++
	return nil
}

type tokenRefreshTestRefresher struct {
	err error
}

func (r *tokenRefreshTestRefresher) CanRefresh(*providercore.Record) bool { return true }

func (r *tokenRefreshTestRefresher) NeedsRefresh(*providercore.Record, time.Duration) bool {
	return true
}

func (r *tokenRefreshTestRefresher) Refresh(context.Context, *providercore.Record) (map[string]any, error) {
	if r.err != nil {
		return nil, r.err
	}
	return map[string]any{"access_token": "new-access-token", "refresh_token": "new-refresh-token"}, nil
}

func TestTokenRefreshService_ProcessRefreshUsesOAuthRefreshCandidates(t *testing.T) {
	future := time.Now().Add(10 * time.Minute)
	repo := &tokenRefreshCandidateRepo{
		providers: []providercore.Record{
			{
				ID:          1,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeOAuth,
				Status:      providercore.StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"refresh_token": "refresh-token"},
			},
			{
				ID:          2,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeOAuth,
				Status:      providercore.StatusActive,
				Schedulable: true,
				Credentials: map[string]any{},
			},
			{
				ID:          3,
				Platform:    capability.PlatformGemini,
				Type:        capability.ProviderTypeAPIKey,
				Status:      providercore.StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"refresh_token": "refresh-token"},
			},
			{
				ID:                      4,
				Platform:                capability.PlatformAntigravity,
				Type:                    capability.ProviderTypeOAuth,
				Status:                  providercore.StatusActive,
				Schedulable:             true,
				Credentials:             map[string]any{"refresh_token": "refresh-token"},
				TempUnschedulableUntil:  &future,
				TempUnschedulableReason: "token refresh retry exhausted: network timeout",
			},
			{
				ID:          5,
				Platform:    "other",
				Type:        capability.ProviderTypeOAuth,
				Status:      providercore.StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"refresh_token": "refresh-token"},
			},
			{
				ID:          6,
				Platform:    capability.PlatformQoder,
				Type:        capability.ProviderTypeCosy,
				Status:      providercore.StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"refresh_token": "refresh-token"},
			},
			{
				ID:                      7,
				Platform:                capability.PlatformAntigravity,
				Type:                    capability.ProviderTypeOAuth,
				Status:                  providercore.StatusActive,
				Schedulable:             true,
				Credentials:             map[string]any{"refresh_token": "refresh-token"},
				Extra:                   map[string]any{"privacy_mode": providercore.AntigravityPrivacySet},
				TempUnschedulableUntil:  &future,
				TempUnschedulableReason: "OAuth 401: unauthorized",
			},
			{
				ID:          8,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeOAuth,
				Status:      providercore.StatusActive,
				Schedulable: false,
				Credentials: map[string]any{"refresh_token": "permanently-rejected-token"},
			},
		},
	}
	tuning := &providercore.RefreshTuning{RefreshBeforeExpiryHours: 1, MaxRetries: 1}
	attempts := backgroundAttemptOptions(repo, tuning)
	post := candidatePostActions(repo)
	attempts.PostActions = post.Run
	svc := providercore.NewBackgroundRefreshService(providercore.BackgroundRefreshOptions{
		Tuning: tuning, Pager: repo, Attempts: attempts,
		Registrations: []providercore.RefreshRegistration{
			{Platform: capability.PlatformOpenAI, Refresher: &tokenRefreshTestRefresher{}},
			{Platform: capability.PlatformGemini, Refresher: &tokenRefreshTestRefresher{}},
			{Platform: capability.PlatformAntigravity, Refresher: &tokenRefreshTestRefresher{}},
			{Platform: capability.PlatformQoder, Refresher: &tokenRefreshTestRefresher{}},
		},
	})

	svc.ScanCycle(context.Background())

	require.Zero(t, repo.listActiveCalls, "TokenRefreshService should not use the broad active-provider query")
	require.ElementsMatch(t, []int64{1, 6, 7}, repo.updatedCredentialIDs)
	require.Equal(t, 1, repo.clearTempCalls, "successful refresh should clear the OAuth 401 temp-unschedulable state")
}

func TestTokenRefreshService_RefreshFailureDoesNotCallPrivacy(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "retry exhausted", err: errors.New("temporary upstream timeout")},
		{name: "non retryable", err: errors.New("invalid_grant: token revoked")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &tokenRefreshCandidateRepo{}
			attempts := backgroundAttemptOptions(repo, &providercore.RefreshTuning{MaxRetries: 1, RetryBackoffSeconds: 0})
			post := candidatePostActions(repo)
			post.Privacy = providercore.NewPrivacyService(nil, nil, PrivacyOptions(func(string) (*req.Client, error) {
				t.Fatalf("privacy client factory must not be called on refresh failure")
				return nil, errors.New("unexpected privacy call")
			}, openai.PrivacyEndpoints{}))
			attempts.PostActions = post.Run
			provider := &providercore.Record{
				ID:       11,
				Platform: capability.PlatformOpenAI,
				Type:     capability.ProviderTypeOAuth,
				Credentials: map[string]any{
					"access_token":  "old-access-token",
					"refresh_token": "refresh-token",
				},
			}

			err := attempts.Run(context.Background(), provider, &tokenRefreshTestRefresher{err: tt.err}, nil, time.Hour, nil)

			require.Error(t, err)
			if IsNonRetryableRefreshError(tt.err) {
				require.Equal(t, 1, repo.setErrorCalls)
				require.Zero(t, repo.setTempUnschedCalls)
			} else {
				require.Zero(t, repo.setErrorCalls)
				require.Equal(t, 1, repo.setTempUnschedCalls)
				require.True(t, strings.HasPrefix(repo.lastTempUnschedReason, "token refresh retry exhausted:"))
			}
		})
	}
}
