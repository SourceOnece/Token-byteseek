//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/ent/account"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/stretchr/testify/require"
)

// 新仓储和旧账号调用共享真实事务：读取顺序、代理、软删除和旧错误码不能因迁移而变化。
func TestProviderMigrationSharesAccountTransaction(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()
	native := providerpostgres.NewStore(client)
	legacy := newAccountRepositoryWithSQL(client, tx, nil)
	p, err := client.Proxy.Create().SetName("migration-proxy").SetProtocol("socks5h").SetHost("proxy.invalid").SetPort(1080).SetUsername("synthetic-user").SetPassword("synthetic-password").Save(ctx)
	require.NoError(t, err)
	zero := 0.0
	root := &provider.Record{Name: "migration-root", Platform: "openai", Type: "oauth", Status: "active", Schedulable: true, AutoPauseOnExpired: true, RateMultiplier: &zero, Concurrency: 3, ProxyID: &p.ID,
		Credentials: map[string]any{"access_token": "synthetic-only", "email": "test@example.invalid"}, Extra: map[string]any{"codex_fingerprint_seed": "00000000-0000-4000-8000-000000000001"}}
	require.NoError(t, native.CreateRecord(ctx, root))
	child := &provider.Record{Name: "migration-shadow", Platform: "openai", Type: "oauth", Status: "active", Schedulable: true, ParentProviderID: &root.ID, QuotaDimension: "spark"}
	require.NoError(t, native.CreateRecord(ctx, child))
	loaded, err := legacy.GetByID(ctx, root.ID)
	require.NoError(t, err)
	require.Equal(t, "synthetic-only", loaded.GetOpenAIAccessToken())
	require.Zero(t, loaded.BillingRateMultiplier())
	require.Equal(t, p.ID, loaded.Proxy.ID)
	require.Equal(t, "synthetic-password", loaded.Proxy.Password)
	// PostgreSQL timestamptz 精度/时区归一后允许微小舍入差异，不能把它误判成版本漂移。
	require.WithinDuration(t, root.UpdatedAt, loaded.UpdatedAt, time.Microsecond)
	batch, err := legacy.GetByIDs(ctx, []int64{child.ID, root.ID, child.ID, -1, 0, 9223372036854775806})
	require.NoError(t, err)
	require.Len(t, batch, 2)
	require.Equal(t, child.ID, batch[0].ID)
	require.Equal(t, root.ID, batch[1].ID)
	require.Equal(t, root.ID, *batch[0].ParentAccountID)
	require.Equal(t, "spark", batch[0].QuotaDimension)
	require.Equal(t, p.ID, batch[1].Proxy.ID)
	_, err = client.Account.Update().Where(account.IDEQ(root.ID)).SetDeletedAt(time.Now()).Save(ctx)
	require.NoError(t, err)
	_, err = native.GetByID(ctx, root.ID)
	require.ErrorIs(t, err, providerpostgres.ErrNotFound)
	_, err = legacy.GetByID(ctx, root.ID)
	require.ErrorIs(t, err, service.ErrAccountNotFound)
	batch, err = legacy.GetByIDs(ctx, []int64{root.ID, child.ID})
	require.NoError(t, err)
	require.Len(t, batch, 1)
}

// 原生写入复用外部事务，清错误不能偷偷开启调度，删除可随事务回滚。
func TestProviderNativeStatePreservesExternalTransaction(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()
	store := providerpostgres.NewStore(client)
	r := &provider.Record{Name: "native-state", Platform: "openai", Type: "oauth", Status: "active", Schedulable: true}
	require.NoError(t, store.CreateRecord(ctx, r))
	require.NoError(t, store.SetErrorRecord(ctx, r.ID, "synthetic"))
	r, err := store.GetByID(ctx, r.ID)
	require.NoError(t, err)
	require.Equal(t, "error", r.Status)
	require.False(t, r.Schedulable)
	require.NoError(t, store.ClearErrorRecord(ctx, r.ID))
	r, err = store.GetByID(ctx, r.ID)
	require.NoError(t, err)
	require.Equal(t, "active", r.Status)
	require.Empty(t, r.ErrorMessage)
	require.False(t, r.Schedulable)
	require.NoError(t, store.SetSchedulableRecord(ctx, r.ID, true))
	r, err = store.GetByID(ctx, r.ID)
	require.NoError(t, err)
	require.True(t, r.Schedulable)
	require.NoError(t, store.DeleteRecord(ctx, r.ID))
	_, err = store.GetByID(ctx, r.ID)
	require.ErrorIs(t, err, providerpostgres.ErrNotFound)
	// 若内部错误提交了调用方事务，下面Rollback将返回ErrTxDone。
	require.NoError(t, tx.Rollback())
	_, err = providerpostgres.NewStore(integrationEntClient).GetByID(ctx, r.ID)
	require.ErrorIs(t, err, providerpostgres.ErrNotFound)
}
