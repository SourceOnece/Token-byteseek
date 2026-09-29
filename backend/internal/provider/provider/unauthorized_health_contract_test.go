//go:build unit

package provider

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func TestRateLimitService_HandleUpstreamError_OAuth401SetsTempUnschedulable(t *testing.T) {
	t.Run("gemini", func(t *testing.T) {
		repo := &unauthorizedHealthStore{}
		invalidator := &unauthorizedTokenRecorder{}
		service := newUnauthorizedObserver(repo, invalidator)
		provider := &providercore.Record{
			ID:       100,
			Platform: capability.PlatformGemini,
			Type:     capability.ProviderTypeOAuth,
			Credentials: map[string]any{
				"refresh_token":              "rt-100",
				"temp_unschedulable_enabled": true,
				"temp_unschedulable_rules": []any{
					map[string]any{
						"error_code":       401,
						"keywords":         []any{"unauthorized"},
						"duration_minutes": 30,
						"description":      "custom rule",
					},
				},
			},
		}

		shouldDisable := service.ApplyUpstreamError(context.Background(), provider, HealthObservation{Status: 401, Body: []byte("unauthorized")}).StopScheduling

		require.True(t, shouldDisable)
		require.Equal(t, 0, repo.setErrorCalls)
		require.Equal(t, 1, repo.tempCalls)
		require.Len(t, invalidator.providers, 1)
	})

	t.Run("antigravity_401_sets_temp_unschedulable", func(t *testing.T) {
		repo := &unauthorizedHealthStore{}
		invalidator := &unauthorizedTokenRecorder{}
		service := newUnauthorizedObserver(repo, invalidator)
		provider := &providercore.Record{
			ID:       100,
			Platform: capability.PlatformAntigravity,
			Type:     capability.ProviderTypeOAuth,
			Status:   billing.StatusActive,
			Credentials: map[string]any{
				"access_token":  "expired-at",
				"refresh_token": "rt-100",
			},
		}

		shouldDisable := service.ApplyUpstreamError(context.Background(), provider, HealthObservation{Status: 401, Body: []byte("unauthorized")}).StopScheduling

		require.True(t, shouldDisable)
		require.Equal(t, 0, repo.setErrorCalls, "Antigravity OAuth 401 must keep status=active so refresh worker can recover it")
		require.Equal(t, 1, repo.tempCalls)
		require.Equal(t, int64(100), repo.lastTempID)
		require.Contains(t, repo.lastTempReason, "invalid or expired credentials")
		require.Equal(t, 1, repo.updateExtraCalls)
		require.Equal(t, true, repo.lastExtraUpdates[providercore.AntigravityForceTokenRefreshExtraKey])
		require.Equal(t, "401_invalid", repo.lastExtraUpdates[providercore.AntigravityForceTokenRefreshReasonExtraKey])
		require.Equal(t, true, provider.Extra[providercore.AntigravityForceTokenRefreshExtraKey])
		require.Len(t, invalidator.providers, 1)
		require.Equal(t, int64(100), invalidator.providers[0].ID)
	})
}

// TestRateLimitService_HandleUpstreamError_SparkShadow401RedirectsToParent 影子无独立凭据,
// 401(母提供商 token 问题)必须重定向到凭据 owner(母提供商)——母提供商 temp-unschedulable + token cache 失效,
// 影子不得被永久禁用(否则母提供商可恢复的 token 问题会把影子永久打死)。
func TestRateLimitService_HandleUpstreamError_SparkShadow401RedirectsToParent(t *testing.T) {
	repo := &unauthorizedHealthStore{}
	repo.providersByID = map[int64]*providercore.Record{}
	invalidator := &unauthorizedTokenRecorder{}
	service := newUnauthorizedObserver(repo, invalidator)

	const parentID = int64(500)
	mother := &providercore.Record{
		ID:          parentID,
		Platform:    capability.PlatformOpenAI,
		Type:        capability.ProviderTypeOAuth,
		Credentials: map[string]any{"refresh_token": "rt-mother"},
	}
	repo.providersByID[parentID] = mother

	shadowParent := parentID
	shadow := &providercore.Record{
		ID:               501,
		Platform:         capability.PlatformOpenAI,
		Type:             capability.ProviderTypeOAuth,
		ParentProviderID: &shadowParent,
		QuotaDimension:   providercore.QuotaDimensionSpark,
		// 影子不持凭据:GetCredential("refresh_token") == ""
	}

	shouldDisable := service.ApplyUpstreamError(context.Background(), shadow, HealthObservation{Status: 401, Body: []byte("unauthorized")}).StopScheduling

	require.True(t, shouldDisable)
	require.Equal(t, 0, repo.setErrorCalls, "spark shadow must not be permanently disabled on a parent-token 401")
	require.Equal(t, 1, repo.tempCalls)
	require.Equal(t, parentID, repo.lastTempID, "temp-unschedulable must target the credential owner (parent)")
	require.Len(t, invalidator.providers, 1)
	require.Equal(t, parentID, invalidator.providers[0].ID, "token cache invalidation must target the parent")
}

// TestRateLimitService_HandleUpstreamError_OAuth401InvalidatorError
// OpenAI OAuth 401 缓存失效出错时仍走 temp_unschedulable。
// 注意：401 handler 不再回写 credentials(避免请求开始时的快照整列覆盖 DB
// 把另一个 worker 刚刷新出来的新 refresh_token 回滚为旧值),
// 因此 updateCredentialsCalls 应当为 0。
func TestRateLimitService_HandleUpstreamError_OAuth401InvalidatorError(t *testing.T) {
	repo := &unauthorizedHealthStore{}
	invalidator := &unauthorizedTokenRecorder{err: errors.New("boom")}
	service := newUnauthorizedObserver(repo, invalidator)
	provider := &providercore.Record{
		ID:       101,
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeOAuth,
		Credentials: map[string]any{
			"refresh_token": "rt-101",
		},
	}

	shouldDisable := service.ApplyUpstreamError(context.Background(), provider, HealthObservation{Status: 401, Body: []byte("unauthorized")}).StopScheduling

	require.True(t, shouldDisable)
	require.Equal(t, 0, repo.setErrorCalls)
	require.Equal(t, 1, repo.tempCalls)
	require.Equal(t, 0, repo.updateCredentialsCalls)
	require.Len(t, invalidator.providers, 1)
}

func TestRateLimitService_HandleUpstreamError_NonOAuth401(t *testing.T) {
	repo := &unauthorizedHealthStore{}
	invalidator := &unauthorizedTokenRecorder{}
	service := newUnauthorizedObserver(repo, invalidator)
	provider := &providercore.Record{
		ID:       102,
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeAPIKey,
	}

	shouldDisable := service.ApplyUpstreamError(context.Background(), provider, HealthObservation{Status: 401, Body: []byte("unauthorized")}).StopScheduling

	require.True(t, shouldDisable)
	require.Equal(t, 1, repo.setErrorCalls)
	require.Empty(t, invalidator.providers)
}

// TestRateLimitService_HandleUpstreamError_OAuth401DoesNotOverwriteCredentials
// 回归测试:确保 401 handler 不再使用请求开始时的 provider 快照写回 credentials。
// 原实现会通过 persistProviderCredentials → UpdateCredentials → SetCredentials
// 整列覆盖 credentials JSONB,在另一个 worker 刚刷新完 refresh_token 的窄窗口内
// 会把新 refresh_token 回滚为快照中的旧值,导致下一周期拿 invalid_grant 被错误 disable。
func TestRateLimitService_HandleUpstreamError_OAuth401DoesNotOverwriteCredentials(t *testing.T) {
	repo := &unauthorizedHealthStore{}
	service := newUnauthorizedObserver(repo, nil)
	provider := &providercore.Record{
		ID:       103,
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeOAuth,
		Credentials: map[string]any{
			"access_token":  "token",
			"refresh_token": "rt-103",
		},
	}

	shouldDisable := service.ApplyUpstreamError(context.Background(), provider, HealthObservation{Status: 401, Body: []byte("unauthorized")}).StopScheduling

	require.True(t, shouldDisable)
	require.Equal(t, 0, repo.updateCredentialsCalls, "401 handler must not write credentials back from the request-start snapshot")
	require.Equal(t, 0, repo.updateExtraCalls, "OpenAI 401 must not set Antigravity force-refresh marker")
	require.Equal(t, 1, repo.tempCalls, "401 handler should still set temp-unschedulable cooldown")
	require.Nil(t, repo.lastCredentials, "no credentials should have been persisted")
}

// 缺失 refresh_token 的 OAuth 提供商 401 后无法靠冷却窗口自愈，应直接标记 error。
func TestRateLimitService_HandleUpstreamError_OAuth401NoRefreshTokenSetsError(t *testing.T) {
	t.Run("openai_no_refresh_token", func(t *testing.T) {
		repo := &unauthorizedHealthStore{}
		invalidator := &unauthorizedTokenRecorder{}
		service := newUnauthorizedObserver(repo, invalidator)
		provider := &providercore.Record{
			ID:       2881,
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeOAuth,
			Credentials: map[string]any{
				"access_token": "expired-at",
			},
		}

		shouldDisable := service.ApplyUpstreamError(context.Background(), provider, HealthObservation{Status: 401, Body: []byte("unauthorized")}).StopScheduling

		require.True(t, shouldDisable)
		require.Equal(t, 1, repo.setErrorCalls)
		require.Equal(t, 0, repo.tempCalls)
		require.Equal(t, 0, repo.updateCredentialsCalls)
		require.Contains(t, repo.lastErrorMsg, "refresh_token missing")
		require.Len(t, invalidator.providers, 1)
	})

	t.Run("blank_refresh_token_treated_as_missing", func(t *testing.T) {
		repo := &unauthorizedHealthStore{}
		service := newUnauthorizedObserver(repo, nil)
		provider := &providercore.Record{
			ID:       2882,
			Platform: capability.PlatformGemini,
			Type:     capability.ProviderTypeOAuth,
			Credentials: map[string]any{
				"access_token":  "expired-at",
				"refresh_token": "   ",
			},
		}

		shouldDisable := service.ApplyUpstreamError(context.Background(), provider, HealthObservation{Status: 401, Body: []byte("unauthorized")}).StopScheduling

		require.True(t, shouldDisable)
		require.Equal(t, 1, repo.setErrorCalls)
		require.Equal(t, 0, repo.tempCalls)
	})

	t.Run("antigravity_no_refresh_token_sets_error", func(t *testing.T) {
		repo := &unauthorizedHealthStore{}
		invalidator := &unauthorizedTokenRecorder{}
		service := newUnauthorizedObserver(repo, invalidator)
		provider := &providercore.Record{
			ID:       2883,
			Platform: capability.PlatformAntigravity,
			Type:     capability.ProviderTypeOAuth,
			Credentials: map[string]any{
				"access_token": "expired-at",
			},
		}

		shouldDisable := service.ApplyUpstreamError(context.Background(), provider, HealthObservation{Status: 401, Body: []byte("unauthorized")}).StopScheduling

		require.True(t, shouldDisable)
		require.Equal(t, 1, repo.setErrorCalls, "Antigravity OAuth without refresh_token cannot self-recover")
		require.Equal(t, 0, repo.tempCalls)
		require.Contains(t, repo.lastErrorMsg, "refresh_token missing")
		require.Len(t, invalidator.providers, 1)
	})
}

// 夹具只组合原生观测入口与窄端口，401 不创建无关平台执行器。
func newUnauthorizedObserver(repo *unauthorizedHealthStore, invalidator *unauthorizedTokenRecorder) *UpstreamHealth {
	options := providercore.HealthOptions{SessionWindows: repo}
	if invalidator != nil {
		options.InvalidateUnauthorizedToken = invalidator.InvalidateToken
	}
	return &UpstreamHealth{Core: providercore.NewHealthService(repo, nil, options)}
}

type unauthorizedHealthStore struct {
	providercore.HealthStore
	providersByID          map[int64]*providercore.Record
	setErrorCalls          int
	tempCalls              int
	updateCredentialsCalls int
	updateExtraCalls       int
	lastCredentials        map[string]any
	lastExtraUpdates       map[string]any
	lastErrorMsg           string
	lastTempUntil          time.Time
	lastTempReason         string
	lastErrorID            int64
	lastTempID             int64
}

func (s *unauthorizedHealthStore) GetByID(_ context.Context, id int64) (*providercore.Record, error) {
	return providercore.CloneRecord(s.providersByID[id]), nil
}

func (s *unauthorizedHealthStore) SetError(_ context.Context, id int64, message string) error {
	s.setErrorCalls++
	s.lastErrorID = id
	s.lastErrorMsg = message
	return nil
}

func (s *unauthorizedHealthStore) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	s.tempCalls++
	s.lastTempID = id
	s.lastTempUntil = until
	s.lastTempReason = reason
	return nil
}

func (s *unauthorizedHealthStore) UpdateExtra(_ context.Context, _ int64, fields map[string]any) error {
	s.updateExtraCalls++
	s.lastExtraUpdates = maps.Clone(fields)
	return nil
}

func (s *unauthorizedHealthStore) UpdateCredentials(_ context.Context, _ int64, fields map[string]any) error {
	s.updateCredentialsCalls++
	s.lastCredentials = maps.Clone(fields)
	return nil
}

func (*unauthorizedHealthStore) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	panic("unexpected window update")
}

type unauthorizedTokenRecorder struct {
	providers []*providercore.Record
	err       error
}

func (s *unauthorizedTokenRecorder) InvalidateToken(_ context.Context, value *providercore.Record) error {
	s.providers = append(s.providers, value)
	return s.err
}
