//go:build integration

package app_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	dbent "github.com/TokenFlux/TokenRouter/ent"
	"github.com/TokenFlux/TokenRouter/internal/app"
	billingpostgres "github.com/TokenFlux/TokenRouter/internal/billing/postgres"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/stretchr/testify/require"
)

// 转接只组合已有存储；资金字段保护和 Ent 事务仍由原生存储执行。
func TestExecutionStoreUsesNativeStateAndOuterTransaction(t *testing.T) {
	f := newDatabaseFixture(t)
	ctx := t.Context()
	data := app.NewProviderStoreForTest(f.client, f.db, nil)
	funds := billingpostgres.NewProviderUsageStore(f.db, billingpostgres.ProviderUsageOptions{})
	store := app.NewExecutionProviderStoreForTest(data, funds)
	row, err := f.client.Provider.Create().SetName("execution-store").SetPlatform(provider.PlatformOpenAI).SetType(provider.ProviderTypeAPIKey).SetCredentials(map[string]any{"api_key": "fixture-key"}).SetExtra(map[string]any{"quota_limit": 100.0, "quota_used": 0.0}).Save(ctx)
	require.NoError(t, err)
	value, err := store.GetByID(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, "fixture-key", value.View().GetCredential("api_key"))
	value.Record.Credentials["api_key"] = "request-private"
	another, err := store.GetByID(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, "fixture-key", another.View().GetCredential("api_key"))
	require.NoError(t, store.IncrementQuotaUsed(ctx, row.ID, 2.5))
	writer, ok := store.(interface {
		UpdateConfiguration(context.Context, *gatewayadapter.ExecutionProvider, provider.ConfigurationChange) error
	})
	require.True(t, ok, "保留执行边界原配置参与能力")
	value.Record.Name = "renamed"
	require.NoError(t, writer.UpdateConfiguration(ctx, value, provider.ConfigurationChange{Fields: provider.ConfigName}))
	persisted, err := data.GetByID(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, "renamed", persisted.Name)
	require.Equal(t, 2.5, persisted.GetQuotaUsed(), "配置不得覆盖读快照后的原子消费")
	require.Equal(t, "fixture-key", persisted.GetCredential("api_key"))
	tx, err := f.client.Tx(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			require.NoError(t, err)
		}
	})
	txctx := dbent.NewTxContext(ctx, tx)
	value.Record.Name = "uncommitted"
	require.NoError(t, writer.UpdateConfiguration(txctx, value, provider.ConfigurationChange{Fields: provider.ConfigName}))
	// 普通查询接口保持原独立读语义；由事务拥有者读取未提交写入以验证同连接参与。
	within, err := tx.Client().Provider.Get(txctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, "uncommitted", within.Name)
	require.NoError(t, tx.Rollback())
	after, err := store.GetByID(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, "renamed", after.Record.Name)
	require.Equal(t, 2.5, after.View().GetQuotaUsed())
}
