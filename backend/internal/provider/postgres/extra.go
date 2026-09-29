package postgres

import (
	"context"
	"encoding/json"
	"errors"
	dbent "github.com/TokenFlux/TokenRouter/ent"
	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// 写操作连同行锁、事务、调度发布与写后缓存一起迁移，保持原调用时机。
func (r *Store) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	discardDeprecatedAccountExtra(updates)
	updates = stripCodexFingerprintSeedFromExtraUpdate(updates)
	if len(updates) == 0 {
		return nil
	}

	// 使用 JSONB 合并操作实现原子更新，避免读-改-写的并发丢失更新问题
	payload, err := json.Marshal(updates)
	if err != nil {
		return err
	}

	durableSchedulerChange := shouldEnqueueSchedulerOutboxForExtraUpdates(updates)
	baseCtx := ctx
	contextTx := dbent.TxFromContext(ctx)
	client := clientFromContext(ctx, r.client)
	var tx *dbent.Tx
	if durableSchedulerChange && contextTx == nil {
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
	extraExpression := "(COALESCE(extra, '{}'::jsonb) - 'upstream_billing_probe' - 'upstream_billing_probe_enabled' - 'openai_long_context_billing_enabled') || $1::jsonb"
	if cnUsageMonitorIdentityExtraPatch(updates) {
		extraExpression = "(" + extraExpression + ") - '" + provider.CNUsageMonitorSnapshotExtraKey + "'"
	}
	if provider.ShouldEnsureFingerprintSeed(updates) {
		extraExpression = ensureCodexFingerprintSeedSQL(extraExpression)
	}
	result, err := client.ExecContext(
		ctx,
		"UPDATE accounts SET extra = "+extraExpression+", updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL",
		string(payload), id,
	)

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
	if durableSchedulerChange {
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
	} else {
		// 观测型 extra 字段不需要触发 bucket 重建，但仍同步单账号快照，
		// 让 sticky session / GetAccount 命中缓存时也能读到最新数据，
		// 同时避免缓存局部 patch 覆盖掉并发写入的其它账号字段。
		if dbent.TxFromContext(ctx) == nil {
			r.syncOne(ctx, id)
		}
	}
	return nil
}
