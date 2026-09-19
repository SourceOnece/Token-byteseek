package repository

import (
	"context"
	"encoding/json"

	"github.com/TokenFlux/TokenRouter/internal/service"
)

// 明确长度结论更新调度：版本/凭据及硬资格校验、账号更新与outbox在同一SQL中完成。
// 不清除限流/到期/质量结果；并发的新关闭或凭据更换优先于迟到采集结果。
func (r *accountRepository) ApplyCodexTicketScheduling(ctx context.Context, a *service.Account, enabled bool) (bool, error) {
	if a == nil || a.Schedulable == enabled {
		return false, nil
	}
	credentials, err := json.Marshal(a.Credentials)
	if err != nil {
		return false, err
	}
	rows, err := r.sql.QueryContext(ctx, `
		WITH changed AS (
			UPDATE accounts SET schedulable=$5,updated_at=NOW()
			WHERE id=$1 AND updated_at=$2 AND credentials=$3::jsonb
			AND deleted_at IS NULL AND schedulable<>$5 AND status='active'
			AND platform='openai' AND type='oauth' AND parent_account_id IS NULL
			AND LOWER(BTRIM(COALESCE(credentials->>'auth_mode',''))) <> 'agentidentity'
			AND (NOT auto_pause_on_expired OR expires_at IS NULL OR expires_at>NOW())
			AND (rate_limit_reset_at IS NULL OR rate_limit_reset_at<=NOW())
			AND (overload_until IS NULL OR overload_until<=NOW())
			AND (temp_unschedulable_until IS NULL OR temp_unschedulable_until<=NOW())
			RETURNING id
		), notification AS (
			INSERT INTO scheduler_outbox(event_type,account_id,group_id,payload)
			SELECT $4,id,NULL,NULL FROM changed
		)
		SELECT EXISTS(SELECT 1 FROM changed)
	`, a.ID, a.UpdatedAt, string(credentials), service.SchedulerOutboxEventAccountChanged, enabled)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return false, rows.Err()
	}
	var changed bool
	if err = rows.Scan(&changed); err != nil {
		return false, err
	}
	if err = rows.Close(); err != nil {
		return false, err
	}
	if changed {
		r.syncSchedulerAccountSnapshot(ctx, a.ID)
	}
	return changed, nil
}
