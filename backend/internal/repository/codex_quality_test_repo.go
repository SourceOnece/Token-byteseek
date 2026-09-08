package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/lib/pq"
)

// AcquireCodexQualityTest 用数据库时间租约跨实例排除同账号重复测试，不覆盖最近结果。
func (r *accountRepository) AcquireCodexQualityTest(ctx context.Context, id int64, runID string) (bool, error) {
	result, err := r.sql.ExecContext(ctx, `
		INSERT INTO codex_quality_tests (account_id,run_id,lease_until)
		SELECT id,$2,NOW()+INTERVAL '150 seconds' FROM accounts
		WHERE id=$1 AND deleted_at IS NULL AND platform='openai' AND type='oauth'
		ON CONFLICT (account_id) DO UPDATE
		SET run_id=EXCLUDED.run_id,lease_until=EXCLUDED.lease_until
		WHERE codex_quality_tests.lease_until <= NOW()
	`, id, runID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

// FinishCodexQualityTest 同一 SQL 原子提交结果、调度开关与 outbox；旧快照或旧租约不能开启账号。
// @project-doc docs/operations/account_maintenance.md#codex_quality_testing
func (r *accountRepository) FinishCodexQualityTest(ctx context.Context, account *service.Account, runID string, result *service.CodexQualityResult) (bool, error) {
	want := result.Status == "full"
	payload, err := json.Marshal(result)
	if err != nil {
		return false, err
	}
	rows, err := r.sql.QueryContext(ctx, `
		WITH owned AS MATERIALIZED (
			SELECT account_id FROM codex_quality_tests
			WHERE account_id=$1 AND run_id=$2 AND lease_until>NOW() FOR UPDATE
		), changed AS (
			UPDATE accounts SET schedulable=$3,updated_at=NOW()
			WHERE id IN (SELECT account_id FROM owned)
			AND deleted_at IS NULL AND platform='openai' AND type='oauth'
			AND updated_at=$4 AND $5 NOT IN ('cancelled','stale')
			RETURNING id,schedulable
		), recorded AS (
			UPDATE codex_quality_tests SET
				result=$6::jsonb || jsonb_build_object(
					'scheduling_applied',EXISTS(SELECT 1 FROM changed),
					'schedulable',COALESCE((SELECT schedulable FROM changed),false),
					'status',CASE WHEN $5='cancelled' THEN 'cancelled'
						WHEN EXISTS(SELECT 1 FROM changed) THEN $5 ELSE 'stale' END),
				lease_until=NOW(),updated_at=NOW()
			WHERE account_id IN (SELECT account_id FROM owned)
			RETURNING result
		), notification AS (
			INSERT INTO scheduler_outbox (event_type,account_id,group_id,payload)
			SELECT $7,id,NULL,NULL FROM changed
		)
		SELECT result FROM recorded
	`, account.ID, runID, want, account.UpdatedAt, result.Status, string(payload), service.SchedulerOutboxEventAccountChanged)
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return false, rows.Err()
	}
	var raw []byte
	if err = rows.Scan(&raw); err != nil {
		return false, err
	}
	if err = json.Unmarshal(raw, result); err != nil {
		return false, err
	}
	// 先结束 SQL 游标，再同步最新账号快照，使已有粘性请求也能看到停调。
	if err = rows.Close(); err != nil {
		return false, err
	}
	if result.SchedulingApplied {
		r.syncSchedulerAccountSnapshot(ctx, account.ID)
	}
	return result.SchedulingApplied, nil
}

// ListCodexQualityResults 只读当前可见账号的最近结果，不将长回答放入调度缓存。
func (r *accountRepository) ListCodexQualityResults(ctx context.Context, ids []int64, detail bool) ([]*service.CodexQualityResult, error) {
	if len(ids) > 500 {
		return nil, errors.New("最多查询 500 个账号")
	}
	rows, err := r.sql.QueryContext(ctx, `SELECT CASE WHEN $2 THEN q.result ELSE q.result - 'prompt' - 'response_text' END FROM codex_quality_tests q
		JOIN accounts a ON a.id=q.account_id
		WHERE a.deleted_at IS NULL AND a.platform='openai' AND a.type='oauth'
		AND q.account_id=ANY($1) AND q.result IS NOT NULL`, pq.Array(ids), detail && len(ids) == 1)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	results := make([]*service.CodexQualityResult, 0)
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		var result service.CodexQualityResult
		if err = json.Unmarshal(raw, &result); err != nil {
			return nil, err
		}
		results = append(results, &result)
	}
	return results, rows.Err()
}
