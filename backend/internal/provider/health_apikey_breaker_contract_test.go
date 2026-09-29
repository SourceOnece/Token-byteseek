package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

type openAIAPIKeyHealthSettingRepo struct {
	providercore.RuntimeSettingsStore
	value    string
	getCalls int
}

func (r *openAIAPIKeyHealthSettingRepo) GetValue(context.Context, string) (string, error) {
	r.getCalls++
	return r.value, nil
}

type openAIAPIKeyHealthProviderRepo struct {
	providercore.HealthStore
	setCalls int
	reason   string
}

func (r *openAIAPIKeyHealthProviderRepo) SetTempUnschedulable(_ context.Context, _ int64, _ time.Time, reason string) error {
	r.setCalls++
	r.reason = reason
	return nil
}

type openAIAPIKeyHealthCacheStub struct {
	providercore.TempUnschedCache
	recordCalls int
	setCalls    int
	tripped     bool
}

func (c *openAIAPIKeyHealthCacheStub) RecordOpenAIAPIKeyHealthFailure(context.Context, int64, int, int) (int64, bool, error) {
	c.recordCalls++
	return 3, c.tripped, nil
}

func (c *openAIAPIKeyHealthCacheStub) SetTempUnsched(context.Context, int64, *providercore.TempUnschedState) error {
	c.setCalls++
	return nil
}

type openAIAPIKeyHealthRuntimeBlocker struct{ calls int }

func (b *openAIAPIKeyHealthRuntimeBlocker) BlockProviderScheduling(*providercore.Record, time.Time, string) {
	b.calls++
}
func (*openAIAPIKeyHealthRuntimeBlocker) ClearProviderSchedulingBlock(int64) {}

func openAIHealthPoolProvider() *providercore.Record {
	return &providercore.Record{
		ID:       42,
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeAPIKey,
		Credentials: map[string]any{
			"pool_mode": true,
		},
	}
}

func TestOpenAIAPIKeyHealthBreakerDefaultDisabled(t *testing.T) {
	settings := providercore.NewRuntimeSettings(&openAIAPIKeyHealthSettingRepo{}, nil)
	cache := &openAIAPIKeyHealthCacheStub{tripped: true}
	svc := providercore.NewHealthService(&openAIAPIKeyHealthProviderRepo{}, cache, providercore.HealthOptions{APIKeyHealthCounter: cache, APIKeyHealthSettings: settings.GetOpenAIAPIKeyHealthBreakerSettings})

	require.False(t, svc.ApplyAPIKeyHealthFailure(context.Background(), openAIHealthPoolProvider(), http.StatusBadGateway, nil, true))
	require.Zero(t, cache.recordCalls)
}

func TestOpenAIAPIKeyHealthBreakerTripsPersistedAndRuntimeState(t *testing.T) {
	encoded, err := json.Marshal(providercore.OpenAIAPIKeyHealthBreakerSettings{Enabled: true, WindowMinutes: 1, FailureThreshold: 3, CooldownMinutes: 5})
	require.NoError(t, err)
	settings := providercore.NewRuntimeSettings(&openAIAPIKeyHealthSettingRepo{value: string(encoded)}, nil)
	cache := &openAIAPIKeyHealthCacheStub{tripped: true}
	repo := &openAIAPIKeyHealthProviderRepo{}
	blocker := &openAIAPIKeyHealthRuntimeBlocker{}
	svc := providercore.NewHealthService(repo, cache, providercore.HealthOptions{APIKeyHealthCounter: cache, APIKeyHealthSettings: settings.GetOpenAIAPIKeyHealthBreakerSettings, Block: blocker.BlockProviderScheduling})
	provider := openAIHealthPoolProvider()

	require.True(t, svc.ApplyAPIKeyHealthFailure(context.Background(), provider, http.StatusBadGateway, []byte(`{"error":"upstream"}`), true))
	require.Equal(t, 1, cache.recordCalls)
	require.Equal(t, 1, cache.setCalls)
	require.Equal(t, 1, repo.setCalls)
	require.Equal(t, 1, blocker.calls)
	require.NotNil(t, provider.TempUnschedulableUntil)
	require.Contains(t, repo.reason, providercore.OpenAIAPIKeyHealthBreakerReason)
}

func TestOpenAIAPIKeyHealthSuccessDoesNotTouchSettingsOrCache(t *testing.T) {
	encoded, err := json.Marshal(providercore.OpenAIAPIKeyHealthBreakerSettings{Enabled: true, WindowMinutes: 1, FailureThreshold: 3, CooldownMinutes: 5})
	require.NoError(t, err)
	settingRepo := &openAIAPIKeyHealthSettingRepo{value: string(encoded)}
	settings := providercore.NewRuntimeSettings(settingRepo, nil)
	cache := &openAIAPIKeyHealthCacheStub{}
	svc := providercore.NewHealthService(&openAIAPIKeyHealthProviderRepo{}, cache, providercore.HealthOptions{APIKeyHealthCounter: cache, APIKeyHealthSettings: settings.GetOpenAIAPIKeyHealthBreakerSettings})

	svc.ObserveAPIKeyHealthSuccess(context.Background(), openAIHealthPoolProvider())
	svc.ObserveAPIKeyHealthSuccess(context.Background(), &providercore.Record{ID: 43, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey})
	require.Zero(t, settingRepo.getCalls)
	require.Zero(t, cache.recordCalls)
}
