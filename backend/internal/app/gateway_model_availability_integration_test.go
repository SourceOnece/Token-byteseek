//go:build integration

package app_test

import (
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/app"
	"github.com/TokenFlux/TokenRouter/internal/protocol"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

// 诊断必须直接读持久配置；瞬时冷却不把已配置模型误报为不存在。
func TestModelAvailabilityUsesPersistentProviderStore(t *testing.T) {
	f := newDatabaseFixture(t)
	ctx := t.Context()
	group, err := f.client.Group.Create().SetName("test-diagnostic-group").SetAllowedProtocols([]protocol.ProtocolID{protocol.ProtocolOpenAIResponses}).Save(ctx)
	require.NoError(t, err)
	cooldown := time.Now().Add(time.Hour)
	row, err := f.client.Provider.Create().SetName("test-diagnostic-provider").SetPlatform(capability.PlatformOpenAI).SetType(capability.ProviderTypeAPIKey).SetCredentials(map[string]any{"model_mapping": map[string]any{"public-known": "public-known"}}).SetRateLimitResetAt(cooldown).SetOverloadUntil(cooldown).SetTempUnschedulableUntil(cooldown).Save(ctx)
	require.NoError(t, err)
	_, err = f.client.ProviderGroup.Create().SetProviderID(row.ID).SetGroupID(group.ID).Save(ctx)
	require.NoError(t, err)
	store := app.NewProviderStoreForTest(f.client, f.db, nil)
	diagnoser := app.NewModelAvailabilityForTest(store, nil)
	t.Run("configured_cooling_provider", func(t *testing.T) {
		result := diagnoser.Compatible.DiagnoseModelAvailabilityForPlatform(ctx, &group.ID, "public-known", capability.PlatformOpenAI)
		require.True(t, result.HasProvidersInPool)
		require.True(t, result.HasModelSupport)
	})
	t.Run("missing_model_with_existing_pool", func(t *testing.T) {
		result := diagnoser.Resolved.DiagnoseModelAvailabilityForPlatform(ctx, &group.ID, "not-configured", capability.PlatformOpenAI)
		require.True(t, result.HasProvidersInPool)
		require.False(t, result.HasModelSupport)
	})
	t.Run("ungrouped_scope", func(t *testing.T) {
		result := diagnoser.Compatible.DiagnoseModelAvailabilityForPlatform(ctx, nil, "public-known", capability.PlatformOpenAI)
		require.False(t, result.HasProvidersInPool)
		require.False(t, result.HasModelSupport)
	})
	t.Run("requires_explicit_group", func(t *testing.T) {
		result := diagnoser.Compatible.DiagnoseModelAvailabilityForPlatform(ctx, nil, "public-known", capability.PlatformOpenAI)
		require.False(t, result.HasProvidersInPool)
		require.False(t, result.HasModelSupport)
		result = diagnoser.Compatible.DiagnoseModelAvailabilityForPlatform(ctx, &group.ID, "public-known", capability.PlatformOpenAI)
		require.True(t, result.HasProvidersInPool)
		require.True(t, result.HasModelSupport)
	})
	// 管理禁用属于持久配置，仍应退出诊断池；不运行健康写入或选号。
	require.NoError(t, f.client.Provider.UpdateOneID(row.ID).SetStatus(provider.StatusDisabled).Exec(ctx))
	result := diagnoser.Compatible.DiagnoseModelAvailabilityForPlatform(ctx, &group.ID, "public-known", capability.PlatformOpenAI)
	require.False(t, result.HasProvidersInPool)
}
