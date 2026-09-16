//go:build integration

package repository

import (
	"context"
	"testing"

	dbent "github.com/TokenFlux/TokenRouter/ent"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/stretchr/testify/require"
)

// 新平台约束必须与注册时的全平台单条批量 INSERT 同步，且不覆盖已有额度。
func TestMiniMaxQuotaMigrationPreservesExistingQuota(t *testing.T) {
	tx := testEntTx(t)
	ctx := dbent.NewTxContext(context.Background(), tx)
	client := tx.Client()
	userID := mustCreateUserForQuota(t, client)
	repo := NewUserPlatformQuotaRepository(client)
	legacyLimit := 42.0
	require.NoError(t, repo.BulkInsertInitial(ctx, []UserPlatformQuotaRecord{{UserID: userID, Platform: service.PlatformOpenAI, DailyLimitUSD: &legacyLimit}}))
	records := make([]UserPlatformQuotaRecord, 0, len(service.AllowedQuotaPlatforms))
	for _, platform := range service.AllowedQuotaPlatforms {
		records = append(records, UserPlatformQuotaRecord{UserID: userID, Platform: platform})
	}
	require.NoError(t, repo.BulkInsertInitial(ctx, records))
	all, err := repo.ListByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, all, len(service.AllowedQuotaPlatforms))
	mini, err := repo.GetByUserPlatform(ctx, userID, service.PlatformMiniMax)
	require.NoError(t, err)
	require.NotNil(t, mini)
	require.Nil(t, mini.DailyLimitUSD)
	existing, err := repo.GetByUserPlatform(ctx, userID, service.PlatformOpenAI)
	require.NoError(t, err)
	require.Equal(t, legacyLimit, *existing.DailyLimitUSD)
	_, err = client.UserPlatformQuota.Create().SetUserID(userID).SetPlatform(service.PlatformMiniMax).Save(ctx)
	require.Error(t, err, "唯一约束仍阻止重复平台额度")
}
