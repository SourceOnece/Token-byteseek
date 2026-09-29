package postgres

import (
	"context"
	"encoding/json"
	"errors"
	dbent "github.com/TokenFlux/TokenRouter/ent"
)

// 写操作连同行锁、事务、调度发布与写后缓存一起迁移，保持原调用时机。
func (r *Store) UpdateCredentials(ctx context.Context, id int64, credentials map[string]any) error {
	payload, err := json.Marshal(normalizeJSONMap(credentials))
	if err != nil {
		return err
	}
	baseCtx := ctx
	contextTx := dbent.TxFromContext(ctx)
	client := r.client
	var tx *dbent.Tx
	if contextTx != nil {
		client = contextTx.Client()
	} else if r.client != nil {
		var txErr error
		tx, txErr = r.client.Tx(ctx)
		if txErr != nil && !errors.Is(txErr, dbent.ErrTxStarted) {
			return txErr
		}
		if tx != nil {
			defer func() { _ = tx.Rollback() }()
			ctx = dbent.NewTxContext(ctx, tx)
			client = tx.Client()
		}
	}
	result, err := client.ExecContext(ctx, `
		UPDATE accounts
		SET
			credentials = $1::jsonb,
			extra = CASE
				-- 凭证整体未变化时不清理 Ollama 状态；废弃账号扩展键始终从写入结果剔除。
				WHEN platform IN ('openai', 'anthropic')
					AND type = 'apikey'
					AND credentials IS DISTINCT FROM $1::jsonb
					AND (
						credentials -> 'api_key' IS DISTINCT FROM $1::jsonb -> 'api_key'
						OR NOT (
							`+ollamaCloudBaseURLMatchesSQL("credentials ->> 'base_url'")+`
							AND `+ollamaCloudBaseURLMatchesSQL("$1::jsonb ->> 'base_url'")+`
						)
					)
				THEN (COALESCE(extra, '{}'::jsonb)
						- 'upstream_billing_probe'
						- 'upstream_billing_probe_enabled'
						- 'openai_long_context_billing_enabled')
					- 'ollama_cloud_usage_session'
					- 'ollama_cloud_usage_auto_refresh'
					- 'ollama_cloud_usage_snapshot'
				ELSE CASE
					WHEN platform IN ('kimi', 'zhipu', 'deepseek')
						AND type = 'apikey'
						AND credentials IS DISTINCT FROM $1::jsonb
					THEN (COALESCE(extra, '{}'::jsonb)
							- 'upstream_billing_probe'
							- 'upstream_billing_probe_enabled'
							- 'openai_long_context_billing_enabled')
						- 'cn_usage_monitor_snapshot'
					ELSE COALESCE(extra, '{}'::jsonb)
						- 'upstream_billing_probe'
						- 'upstream_billing_probe_enabled'
						- 'openai_long_context_billing_enabled'
				END
			END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`, string(payload), id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return r.notFound()
	}
	if err := r.publish(ctx, client, eventChanged, &id, nil); err != nil {
		return err
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	if contextTx == nil {
		r.syncOne(baseCtx, id)
	}
	return nil
}
