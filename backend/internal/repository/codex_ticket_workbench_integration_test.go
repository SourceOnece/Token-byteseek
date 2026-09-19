//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	dbaccount "github.com/TokenFlux/TokenRouter/ent/account"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCodexTicketAttemptAndCooldownRedis(t *testing.T) {
	ctx := context.Background()
	rdb := testRedis(t)
	cache := NewCodexTicketCache(rdb).(*codexTicketCache)
	for i := 1; i <= 4; i++ {
		n, err := cache.NextTicketAttempt(ctx, "unlimited", 0)
		require.NoError(t, err)
		require.Equal(t, i, n)
	}
	for i := 1; i <= 3; i++ {
		n, err := cache.NextTicketAttempt(ctx, "limited", 2)
		require.NoError(t, err)
		if i > 2 {
			require.Zero(t, n)
		} else {
			require.Equal(t, i, n)
		}
	}
	now := time.Now()
	require.NoError(t, cache.RecordTicketCollection(ctx, "unlimited", false, 2, 60, 0, now))
	require.NoError(t, cache.RecordTicketCollection(ctx, "unlimited", false, 2, 60, 0, now))
	raw, err := cache.Get(ctx, "collection:unlimited")
	require.NoError(t, err)
	var state struct {
		Failures int   `json:"failures"`
		Until    int64 `json:"until"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &state))
	require.Equal(t, 2, state.Failures)
	require.Greater(t, state.Until, now.UnixMilli())
	n, err := cache.NextTicketAttempt(ctx, "unlimited", 0)
	require.NoError(t, err)
	require.Equal(t, 5, n, "不限模式跨冷却保留累计次数")
	require.NoError(t, cache.RecordTicketCollection(ctx, "unlimited", true, 2, 60, 0, now))
	raw, err = cache.Get(ctx, "collection:unlimited")
	require.NoError(t, err)
	require.Empty(t, raw)
}

// 筛选直接作用Count与分页，且不要求把代理密文复制到账号表。
func TestCodexTicketFiltersRealDatabase(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	settings := NewSettingRepository(client)
	a, err := client.Account.Create().SetName("ticket-a").SetPlatform("openai").SetType("oauth").SetCredentials(map[string]any{}).Save(ctx)
	require.NoError(t, err)
	b, err := client.Account.Create().SetName("ticket-b").SetPlatform("openai").SetType("oauth").SetCredentials(map[string]any{}).Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = settings.Delete(ctx, "codex_ticket_runtime")
		_, _ = client.Account.Delete().Where(dbaccount.IDIn(a.ID, b.ID)).Exec(ctx)
	})
	raw := fmt.Sprintf(`{"enabled":true,"target_length":292,"selection_mode":"fixed","accounts":{"%d":{"mode":"off","rules":{"target_length":332},"proxy_policy":{"mode":"dynamic"}},"%d":{"mode":"on","rules":{"target_length":356}}}}`, a.ID, b.ID)
	require.NoError(t, settings.Set(ctx, "codex_ticket_runtime", raw))
	for filter, want := range map[string]int{"on": 1, "off": 1, "length:292": 0, "length:332": 1, "length:356": 1, "dynamic": 1, "proxy_account": 1, "proxy_gateway": 1, "configured": 2} {
		q := client.Account.Query().Where(dbaccount.IDIn(a.ID, b.ID))
		applyCodexTicketFilter(q, filter)
		count, err := q.Clone().Count(ctx)
		require.NoError(t, err, filter)
		require.Equal(t, want, count, filter)
		items, err := q.Limit(1).All(ctx)
		require.NoError(t, err)
		if want > 0 {
			require.Len(t, items, 1)
		}
	}
	require.False(t, service.ValidCodexTicketFilter("length:1 OR 1=1"))
}
