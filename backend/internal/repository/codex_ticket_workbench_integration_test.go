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
	// 实际长度只使用服务端解析出的匹配集合，并且先筛选再Count/Offset。
	q := client.Account.Query().Where(dbaccount.IDIn(a.ID, b.ID))
	applyCodexTicketFilter(q, "on,actual_length:332", []int64{a.ID, b.ID})
	count, err := q.Clone().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	items, err := q.Limit(1).All(ctx)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, b.ID, items[0].ID)
	unresolved := client.Account.Query().Where(dbaccount.IDIn(a.ID, b.ID))
	applyCodexTicketFilter(unresolved, "actual_length:332")
	count, err = unresolved.Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count, "缺少解析结果不能放行全部账号")
	// 旧号无独立规则时，静态兼容筛选与运行态一样读取冻结快照，不读新号模板。
	normalized := fmt.Sprintf(`{"enabled":true,"target_length":292,"legacy_account_defaults":{"rules":{"target_length":444}},"import_defaults":{"rules":{"target_length":512}},"accounts":{"%d":{"mode":"on","rules":{"target_length":356}}}}`, b.ID)
	require.NoError(t, settings.Set(ctx, "codex_ticket_runtime", normalized))
	for filter, want := range map[string]int{"length:444": 1, "length:356": 1, "length:292": 0, "length:512": 0} {
		query := client.Account.Query().Where(dbaccount.IDIn(a.ID, b.ID))
		applyCodexTicketFilter(query, filter)
		count, e := query.Count(ctx)
		require.NoError(t, e)
		require.Equal(t, want, count, filter)
	}
}
