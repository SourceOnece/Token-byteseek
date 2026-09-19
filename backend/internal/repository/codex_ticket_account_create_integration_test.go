//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"github.com/TokenFlux/TokenRouter/ent/account"
	"github.com/TokenFlux/TokenRouter/ent/setting"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
)

// 真实事务验证配置冲突/分组错误回滚账号，成功号与私有配置一起可见。
func TestCodexTicketAccountCreationAtomic(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := newAccountRepositoryWithSQL(client, integrationDB, nil)
	settings := NewSettingRepository(client)
	raw := `{"enabled":false,"generation":"synthetic-import","accounts":{}}`
	old, readErr := settings.GetValue(ctx, "codex_ticket_runtime")
	t.Cleanup(func() {
		if readErr == nil {
			_ = settings.Set(ctx, "codex_ticket_runtime", old)
		} else {
			_, _ = client.Setting.Delete().Where(setting.KeyEQ("codex_ticket_runtime")).Exec(ctx)
		}
	})
	require.NoError(t, settings.Set(ctx, "codex_ticket_runtime", raw))
	makeAccount := func(suffix string) *service.Account {
		return &service.Account{Name: uniqueTestValue(t, "ticket-import-"+suffix), Platform: "openai", Type: "oauth", Status: "active", Schedulable: true, Credentials: map[string]any{"access_token": "synthetic"}, Concurrency: 1}
	}
	a := makeAccount("success")
	in := service.CodexTicketAccountCreation{Expected: &raw, Configuration: raw, AccountConfiguration: `{"mode":"on","revision":"new","rules":{"target_length":332}}`}
	require.NoError(t, repo.CreateWithCodexTicket(ctx, a, nil, in))
	stored, err := settings.GetValue(ctx, "codex_ticket_runtime")
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(stored), &parsed))
	require.Len(t, parsed["accounts"], 1)
	b := makeAccount("conflict")
	// 第二次使用旧 expected 应冲突；检查以旧配置快照前的内容为准。
	require.Error(t, repo.CreateWithCodexTicket(ctx, b, nil, service.CodexTicketAccountCreation{Expected: &raw, Configuration: raw, AccountConfiguration: in.AccountConfiguration}))
	count, err := client.Account.Query().Where(account.NameEQ(b.Name)).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
	in.Expected = &stored
	in.Configuration = stored
	c := makeAccount("group-error")
	require.Error(t, repo.CreateWithCodexTicket(ctx, c, []int64{999999999}, in))
	count, err = client.Account.Query().Where(account.NameEQ(c.Name)).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}
