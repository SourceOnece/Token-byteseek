package postgres

import (
	"context"
	"encoding/json"
	"errors"
	dbent "github.com/TokenFlux/TokenRouter/ent"
	dbaccount "github.com/TokenFlux/TokenRouter/ent/account"
	"github.com/TokenFlux/TokenRouter/internal/domain"
	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// 写操作连同行锁、事务、调度发布与写后缓存一起迁移，保持原调用时机。
func (r *Store) Update(ctx context.Context, account *provider.Record) error {
	if account == nil {
		return nil
	}

	baseCtx := ctx
	contextTx := dbent.TxFromContext(ctx)
	client := r.client
	var tx *dbent.Tx
	if contextTx != nil {
		client = contextTx.Client()
	} else {
		var err error
		tx, err = r.client.Tx(ctx)
		if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
			return err
		}
		if tx != nil {
			defer func() { _ = tx.Rollback() }()
			ctx = dbent.NewTxContext(ctx, tx)
			client = tx.Client()
		}
	}

	updated, err := r.updateLockedRecord(ctx, client, account)
	if err != nil {
		return r.translateError(err)
	}
	if err := r.publish(ctx, client, eventChanged, &account.ID, buildSchedulerGroupPayload(account.GroupIDs)); err != nil {
		return err
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	account.UpdatedAt = updated.UpdatedAt
	// 普通账号编辑（如 model_mapping / credentials）也需要立即刷新单账号快照，
	// 否则网关在 outbox worker 延迟或异常时仍可能读到旧配置。
	if contextTx == nil {
		r.syncOne(baseCtx, account.ID)
	}
	return nil
}

func (r *Store) updateLockedRecord(ctx context.Context, client *dbent.Client, account *provider.Record) (*dbent.Account, error) {
	extra, err := r.LockAndMergeManagedExtra(ctx, client, account)
	if err != nil {
		return nil, err
	}
	account.Extra = extra

	schedulable := account.Schedulable
	if account.Status == domain.StatusError {
		// 错误状态账号必须退出调度池，避免后台更新把失效账号重新放回可用列表。
		schedulable = false
	}

	builder := client.Account.UpdateOneID(account.ID).
		SetName(account.Name).
		SetNillableNotes(account.Notes).
		SetPlatform(account.Platform).
		SetType(account.Type).
		SetCredentials(normalizeJSONMap(account.Credentials)).
		SetExtra(extra).
		SetConcurrency(account.Concurrency).
		SetPriority(account.Priority).
		SetStatus(account.Status).
		SetErrorMessage(account.ErrorMessage).
		SetSchedulable(schedulable).
		SetAutoPauseOnExpired(account.AutoPauseOnExpired)

	if account.RateMultiplier != nil {
		builder.SetRateMultiplier(*account.RateMultiplier)
	}
	if account.LoadFactor != nil {
		builder.SetLoadFactor(*account.LoadFactor)
	} else {
		builder.ClearLoadFactor()
	}

	if account.ProxyID != nil {
		builder.SetProxyID(*account.ProxyID)
	} else {
		builder.ClearProxyID()
	}
	if account.LastUsedAt != nil {
		builder.SetLastUsedAt(*account.LastUsedAt)
	} else {
		builder.ClearLastUsedAt()
	}
	if account.ExpiresAt != nil {
		builder.SetExpiresAt(*account.ExpiresAt)
	} else {
		builder.ClearExpiresAt()
	}
	if account.RateLimitedAt != nil {
		builder.SetRateLimitedAt(*account.RateLimitedAt)
	} else {
		builder.ClearRateLimitedAt()
	}
	if account.RateLimitResetAt != nil {
		builder.SetRateLimitResetAt(*account.RateLimitResetAt)
	} else {
		builder.ClearRateLimitResetAt()
	}
	if account.OverloadUntil != nil {
		builder.SetOverloadUntil(*account.OverloadUntil)
	} else {
		builder.ClearOverloadUntil()
	}
	if account.SessionWindowStart != nil {
		builder.SetSessionWindowStart(*account.SessionWindowStart)
	} else {
		builder.ClearSessionWindowStart()
	}
	if account.SessionWindowEnd != nil {
		builder.SetSessionWindowEnd(*account.SessionWindowEnd)
	} else {
		builder.ClearSessionWindowEnd()
	}
	if account.SessionWindowStatus != "" {
		builder.SetSessionWindowStatus(account.SessionWindowStatus)
	} else {
		builder.ClearSessionWindowStatus()
	}
	if account.Notes == nil {
		builder.ClearNotes()
	}

	builder.SetQuotaDimension(dbaccount.QuotaDimension(quotaDimension(account.QuotaDimension)))
	builder.SetNillableParentAccountID(account.ParentProviderID)

	return builder.Save(ctx)
}

func (r *Store) LockAndMergeManagedExtra(ctx context.Context, client *dbent.Client, account *provider.Record) (map[string]any, error) {
	credentials, err := json.Marshal(normalizeJSONMap(account.Credentials))
	if err != nil {
		return nil, err
	}
	var proxyID any
	if account.ProxyID != nil {
		proxyID = *account.ProxyID
	}
	rows, err := client.QueryContext(ctx, `
		SELECT
			COALESCE(
				platform IN ('openai', 'anthropic')
				AND $2 IN ('openai', 'anthropic')
				AND type = 'apikey'
				AND $3 = 'apikey'
				AND credentials -> 'api_key' IS NOT DISTINCT FROM $4::jsonb -> 'api_key'
				AND `+ollamaCloudBaseURLMatchesSQL("credentials ->> 'base_url'")+`
				AND `+ollamaCloudBaseURLMatchesSQL("$4::jsonb ->> 'base_url'")+`,
				false
			),
			proxy_id IS NOT DISTINCT FROM $5,
			extra -> 'ollama_cloud_usage_session',
			extra -> 'ollama_cloud_usage_auto_refresh',
			extra -> 'ollama_cloud_usage_snapshot'
		FROM accounts
		WHERE id = $1 AND deleted_at IS NULL
		FOR NO KEY UPDATE
	`, account.ID, account.Platform, account.Type, string(credentials), proxyID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, r.notFound()
	}

	var (
		ollamaGroupIdentityUnchanged bool
		ollamaProxyIdentityUnchanged bool
		currentOllamaSession         []byte
		currentOllamaAutoRefresh     []byte
		currentOllamaSnapshot        []byte
	)
	if err := rows.Scan(
		&ollamaGroupIdentityUnchanged,
		&ollamaProxyIdentityUnchanged,
		&currentOllamaSession,
		&currentOllamaAutoRefresh,
		&currentOllamaSnapshot,
	); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	extra := copyJSONMap(normalizeJSONMap(account.Extra))
	discardDeprecatedAccountExtra(extra)
	for _, key := range []string{
		provider.OllamaCloudUsageSessionExtraKey,
		provider.OllamaCloudUsageAutoRefreshExtraKey,
		provider.OllamaCloudUsageSnapshotExtraKey,
	} {
		delete(extra, key)
	}
	if provider.IsOllamaCloudUsageRecord(account) && ollamaGroupIdentityUnchanged {
		for key, raw := range map[string][]byte{
			provider.OllamaCloudUsageSessionExtraKey:     currentOllamaSession,
			provider.OllamaCloudUsageAutoRefreshExtraKey: currentOllamaAutoRefresh,
		} {
			if value, ok, err := decodeAccountExtraJSON(raw); err != nil {
				return nil, err
			} else if ok {
				extra[key] = value
			}
		}
		if ollamaProxyIdentityUnchanged {
			if snapshot, ok, err := decodeAccountExtraJSON(currentOllamaSnapshot); err != nil {
				return nil, err
			} else if ok {
				extra[provider.OllamaCloudUsageSnapshotExtraKey] = snapshot
			}
		}
	}
	return extra, nil
}
