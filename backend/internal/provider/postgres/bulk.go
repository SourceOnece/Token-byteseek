package postgres

import (
	"context"
	"encoding/json"
	"errors"
	dbent "github.com/TokenFlux/TokenRouter/ent"
	"github.com/TokenFlux/TokenRouter/internal/domain"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/lib/pq"
)

// 写操作连同行锁、事务、调度发布与写后缓存一起迁移，保持原调用时机。
func (r *Store) BulkUpdate(ctx context.Context, ids []int64, updates provider.BulkUpdate) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	discardDeprecatedAccountExtra(updates.Extra)
	updates.Extra = stripCodexFingerprintSeedFromExtraUpdate(updates.Extra)

	setClauses := make([]string, 0, 8)
	args := make([]any, 0, 8)

	idx := 1
	ollamaProxyIdentityChanged := ""
	// 批量未提供保持；显式空备注和零到期分别清空，避免null丢失清除意图。
	if updates.Notes != nil {
		setClauses = append(setClauses, "notes = $"+itoa(idx))
		args = append(args, *updates.Notes)
		idx++
	}
	if updates.ClearExpiresAt {
		setClauses = append(setClauses, "expires_at = NULL")
	} else if updates.ExpiresAt != nil {
		setClauses = append(setClauses, "expires_at = $"+itoa(idx))
		args = append(args, *updates.ExpiresAt)
		idx++
	}
	if updates.AutoPauseOnExpired != nil {
		setClauses = append(setClauses, "auto_pause_on_expired = $"+itoa(idx))
		args = append(args, *updates.AutoPauseOnExpired)
		idx++
	}
	if updates.Name != nil {
		setClauses = append(setClauses, "name = $"+itoa(idx))
		args = append(args, *updates.Name)
		idx++
	}
	if updates.ProxyID != nil {
		// 0 表示清除代理（前端发送 0 而不是 null 来表达清除意图）
		if *updates.ProxyID == 0 {
			setClauses = append(setClauses, "proxy_id = NULL")
			ollamaProxyIdentityChanged = "proxy_id IS NOT NULL"
		} else {
			proxyPlaceholder := "$" + itoa(idx)
			setClauses = append(setClauses, "proxy_id = "+proxyPlaceholder)
			ollamaProxyIdentityChanged = "proxy_id IS DISTINCT FROM " + proxyPlaceholder
			args = append(args, *updates.ProxyID)
			idx++
		}
	}
	if updates.Concurrency != nil {
		setClauses = append(setClauses, "concurrency = $"+itoa(idx))
		args = append(args, *updates.Concurrency)
		idx++
	}
	if updates.Priority != nil {
		setClauses = append(setClauses, "priority = $"+itoa(idx))
		args = append(args, *updates.Priority)
		idx++
	}
	if updates.RateMultiplier != nil {
		setClauses = append(setClauses, "rate_multiplier = $"+itoa(idx))
		args = append(args, *updates.RateMultiplier)
		idx++
	}
	if updates.LoadFactor != nil {
		if *updates.LoadFactor <= 0 {
			setClauses = append(setClauses, "load_factor = NULL")
		} else {
			setClauses = append(setClauses, "load_factor = $"+itoa(idx))
			args = append(args, *updates.LoadFactor)
			idx++
		}
	}
	if updates.Status != nil {
		setClauses = append(setClauses, "status = $"+itoa(idx))
		args = append(args, *updates.Status)
		idx++
	}
	if updates.Schedulable != nil {
		setClauses = append(setClauses, "schedulable = $"+itoa(idx))
		args = append(args, *updates.Schedulable)
		idx++
	}
	// JSONB 需要合并而非覆盖，使用 raw SQL 保持旧行为。
	credentialPlaceholder := ""
	if len(updates.Credentials) > 0 {
		payload, err := json.Marshal(updates.Credentials)
		if err != nil {
			return 0, err
		}
		credentialPlaceholder = "$" + itoa(idx)
		setClauses = append(setClauses, "credentials = COALESCE(credentials, '{}'::jsonb) || "+credentialPlaceholder+"::jsonb")
		args = append(args, payload)
		idx++
	}

	ollamaGroupIdentityChanges := make([]string, 0, 2)
	if _, ok := updates.Credentials["api_key"]; ok {
		ollamaGroupIdentityChanges = append(ollamaGroupIdentityChanges, "credentials -> 'api_key' IS DISTINCT FROM "+credentialPlaceholder+"::jsonb -> 'api_key'")
	}
	if _, ok := updates.Credentials["base_url"]; ok {
		ollamaGroupIdentityChanges = append(ollamaGroupIdentityChanges,
			"NOT ("+ollamaCloudBaseURLMatchesSQL("credentials ->> 'base_url'")+
				" AND "+ollamaCloudBaseURLMatchesSQL(credentialPlaceholder+"::jsonb ->> 'base_url'")+")")
	}

	cnUsageIdentityChanged := len(updates.Credentials) > 0 || updates.ProxyID != nil || cnUsageMonitorIdentityExtraPatch(updates.Extra)
	if len(updates.Extra) > 0 || len(ollamaGroupIdentityChanges) > 0 || ollamaProxyIdentityChanged != "" || cnUsageIdentityChanged || updates.EnsureCodexFingerprintSeed {
		extraExpression := "COALESCE(extra, '{}'::jsonb) - 'upstream_billing_probe' - 'upstream_billing_probe_enabled' - 'openai_long_context_billing_enabled'"
		if len(updates.Extra) > 0 {
			payload, err := json.Marshal(updates.Extra)
			if err != nil {
				return 0, err
			}
			extraExpression += " || $" + itoa(idx) + "::jsonb"
			args = append(args, payload)
			idx++
			if ollamaCloudUsageSnapshotClearRequested(updates.Extra) {
				extraExpression = "(" + extraExpression + ") - 'ollama_cloud_usage_snapshot'"
			}
		}
		eligibleAccount := "platform IN ('openai', 'anthropic') AND type = 'apikey'"
		groupIdentityChanged := ""
		if len(ollamaGroupIdentityChanges) > 0 {
			groupIdentityChanged = "(" + eligibleAccount + " AND (" + joinClauses(ollamaGroupIdentityChanges, " OR ") + "))"
		}
		snapshotIdentityChanged := groupIdentityChanged
		if ollamaProxyIdentityChanged != "" {
			proxyChanged := "(" + eligibleAccount + " AND " + ollamaProxyIdentityChanged + ")"
			if snapshotIdentityChanged == "" {
				snapshotIdentityChanged = proxyChanged
			} else {
				snapshotIdentityChanged = "(" + snapshotIdentityChanged + " OR " + proxyChanged + ")"
			}
		}
		if groupIdentityChanged != "" {
			extraExpression = "CASE" +
				" WHEN " + groupIdentityChanged + " THEN (" + extraExpression + ") - 'ollama_cloud_usage_session' - 'ollama_cloud_usage_auto_refresh' - 'ollama_cloud_usage_snapshot'" +
				" WHEN " + snapshotIdentityChanged + " THEN (" + extraExpression + ") - 'ollama_cloud_usage_snapshot'" +
				" ELSE " + extraExpression + " END"
		} else if snapshotIdentityChanged != "" {
			extraExpression = "CASE WHEN " + snapshotIdentityChanged + " THEN (" + extraExpression + ") - 'ollama_cloud_usage_snapshot' ELSE " + extraExpression + " END"
		}
		if cnUsageIdentityChanged {
			extraExpression = "CASE WHEN platform IN ('kimi', 'zhipu', 'deepseek') AND type = 'apikey'" +
				" THEN (" + extraExpression + ") - '" + provider.CNUsageMonitorSnapshotExtraKey + "' ELSE " + extraExpression + " END"
		}
		if updates.EnsureCodexFingerprintSeed {
			extraExpression = ensureCodexFingerprintSeedSQL(extraExpression)
		}
		setClauses = append(setClauses, "extra = "+extraExpression)
	}

	if len(setClauses) == 0 {
		return 0, nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	whereClause := " WHERE id = ANY($" + itoa(idx) + ") AND deleted_at IS NULL"
	args = append(args, pq.Array(ids))
	query := "UPDATE accounts SET " + joinClauses(setClauses, ", ") + whereClause

	baseCtx := ctx
	contextTx := dbent.TxFromContext(ctx)
	exec := r.sql
	var tx *dbent.Tx
	if contextTx != nil {
		exec = contextTx.Client()
	} else if r.client != nil {
		var txErr error
		tx, txErr = r.client.Tx(ctx)
		if txErr != nil && !errors.Is(txErr, dbent.ErrTxStarted) {
			return 0, txErr
		}
		if tx != nil {
			defer func() { _ = tx.Rollback() }()
			ctx = dbent.NewTxContext(ctx, tx)
			exec = tx.Client()
		}
	}

	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows > 0 {
		payload := map[string]any{"account_ids": ids}
		if err := r.publish(ctx, exec, eventBulkChanged, nil, payload); err != nil {
			return 0, err
		}
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return 0, err
		}
	}
	if rows > 0 && contextTx == nil {
		shouldSync := false
		if updates.Status != nil && (*updates.Status == domain.StatusError || *updates.Status == domain.StatusDisabled) {
			shouldSync = true
		}
		if updates.Schedulable != nil && !*updates.Schedulable {
			shouldSync = true
		}
		if shouldSync {
			r.syncMany(baseCtx, ids)
		}
	}
	return rows, nil
}
