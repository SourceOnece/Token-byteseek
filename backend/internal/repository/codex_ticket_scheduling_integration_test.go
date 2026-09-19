//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/stretchr/testify/require"
)

// 真实SQL验证开/关、幂等、版本/凭据保护和outbox原子写入；不访问真实上游。
func TestCodexTicketSchedulingRealDatabase(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := newAccountRepositoryWithSQL(client, integrationDB, nil)
	a := mustCreateAccount(t, client, &service.Account{Name: "synthetic-ticket-scheduling", Platform: "openai", Type: "oauth", Status: "active", Credentials: map[string]any{"access_token": "synthetic"}})
	count := func() int {
		var n int
		require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT count(*) FROM scheduler_outbox WHERE account_id=$1", a.ID).Scan(&n))
		return n
	}
	for _, enabled := range []bool{false, true} {
		before, err := repo.GetByID(ctx, a.ID)
		require.NoError(t, err)
		n := count()
		changed, err := repo.ApplyCodexTicketScheduling(ctx, before, enabled)
		require.NoError(t, err)
		require.False(t, changed.IsZero())
		require.Equal(t, n+1, count())
		after, err := repo.GetByID(ctx, a.ID)
		require.NoError(t, err)
		require.Equal(t, enabled, after.Schedulable)
		require.True(t, changed.Equal(after.UpdatedAt), "返回精确数据库版本，不能用进程时间代替")
		changed, err = repo.ApplyCodexTicketScheduling(ctx, after, enabled)
		require.NoError(t, err)
		require.True(t, changed.IsZero())
		require.Equal(t, n+1, count())
	}
	// 保存之后出现的手工修改使旧版本失效，不允许迟到结果覆盖。
	before, err := repo.GetByID(ctx, a.ID)
	require.NoError(t, err)
	_, err = client.Account.UpdateOneID(a.ID).SetName("newer-settings").Save(ctx)
	require.NoError(t, err)
	n := count()
	changed, err := repo.ApplyCodexTicketScheduling(ctx, before, false)
	require.NoError(t, err)
	require.True(t, changed.IsZero())
	require.Equal(t, n, count())
	// 即使调用者传入最新版，当前上游限流也不能被票据结论清除。
	_, err = client.Account.UpdateOneID(a.ID).SetSchedulable(false).SetRateLimitResetAt(time.Now().Add(time.Hour)).Save(ctx)
	require.NoError(t, err)
	before, err = repo.GetByID(ctx, a.ID)
	require.NoError(t, err)
	changed, err = repo.ApplyCodexTicketScheduling(ctx, before, true)
	require.NoError(t, err)
	require.True(t, changed.IsZero())
	require.Equal(t, n, count())
	_, err = client.Account.UpdateOneID(a.ID).ClearRateLimitResetAt().Save(ctx)
	require.NoError(t, err)
	before, err = repo.GetByID(ctx, a.ID)
	require.NoError(t, err)
	before.Credentials = map[string]any{"access_token": "old-synthetic"}
	changed, err = repo.ApplyCodexTicketScheduling(ctx, before, true)
	require.NoError(t, err)
	require.True(t, changed.IsZero())
	require.Equal(t, n, count())
}
