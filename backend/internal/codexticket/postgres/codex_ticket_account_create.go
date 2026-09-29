package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/TokenFlux/TokenRouter/ent/setting"
	"github.com/TokenFlux/TokenRouter/internal/codexticket"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
)

// 同一事务创建账号、分组、私有票据和outbox。配置冲突/失败会回滚新号，不留下半成功账号。
func (r *Store) CreateWithCodexTicket(ctx context.Context, a *codexticket.Account, groups []int64, input codexticket.CodexTicketAccountCreation) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()
	if err := providerpostgres.CreateRecord(ctx, client, a); err != nil {
		return err
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal([]byte(input.Configuration), &cfg); err != nil {
		return err
	}
	accounts := map[string]json.RawMessage{}
	if raw := cfg["accounts"]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &accounts); err != nil {
			return err
		}
	}
	if accounts == nil {
		accounts = map[string]json.RawMessage{}
	}
	accounts[strconv.FormatInt(a.ID, 10)] = json.RawMessage(input.AccountConfiguration)
	cfg["accounts"], err = json.Marshal(accounts)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	const key = "codex_ticket_runtime"
	if input.Expected == nil {
		_, err = client.Setting.Create().SetKey(key).SetValue(string(raw)).Save(ctx)
	} else {
		var n int
		n, err = client.Setting.Update().Where(setting.KeyEQ(key), setting.ValueEQ(*input.Expected)).SetValue(string(raw)).SetUpdatedAt(time.Now()).Save(ctx)
		if err == nil && n != 1 {
			err = fmt.Errorf("票据配置已变化，请重试创建；本次未创建账号")
		}
	}
	if err != nil {
		return err
	}
	seen := map[int64]bool{}
	for _, id := range groups {
		if seen[id] {
			continue
		}
		seen[id] = true
		if _, err := client.ProviderGroup.Create().SetProviderID(a.ID).SetGroupID(id).Save(ctx); err != nil {
			return err
		}
	}
	a.GroupIDs = append([]int64(nil), groups...)
	if err := r.events.Write(ctx, client, providerpostgres.ProviderChanged, &a.ID, nil, r.events.GroupPayload(groups)); err != nil {
		return err
	}
	return tx.Commit()
}
