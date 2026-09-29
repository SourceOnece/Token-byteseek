package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TokenFlux/TokenRouter/internal/quality"
)

// 计划、轮次和回答分表；列表始终只返回摘要，正文按分类分页读取。
func (r *Store) SaveQualitySchedule(ctx context.Context, p *quality.CodexQualitySchedule) error {
	config, err := json.Marshal(p.Config)
	if err != nil {
		return err
	}
	var query string
	var args []any
	if p.ID == 0 {
		query = `INSERT INTO codex_quality_schedules(name,interval_minutes,keep_runs,enabled,config,next_run_at)
		VALUES($1,$2,$3,$4,$5,NOW()+make_interval(mins=>$2)) RETURNING id`
		args = []any{p.Name, p.IntervalMinutes, p.KeepRuns, p.Enabled, string(config)}
	} else {
		// 编辑立即撤销旧运行的提交资格；旧 worker 下一次续租时取消。
		query = `WITH locked AS MATERIALIZED (SELECT id,active_run_id FROM codex_quality_schedules WHERE id=$6 FOR UPDATE),
		 old AS (UPDATE codex_quality_runs SET status='interrupted',finished_at=NOW()
		 WHERE id=(SELECT active_run_id FROM locked) AND status='running')
		 UPDATE codex_quality_schedules SET name=$1,interval_minutes=$2,keep_runs=$3,enabled=$4,config=$5,
		 next_run_at=NOW()+make_interval(mins=>$2),active_run_id=NULL,lease_until=NULL,manual_requested_at=NULL,updated_at=NOW()
		 WHERE id IN(SELECT id FROM locked) RETURNING id`
		args = []any{p.Name, p.IntervalMinutes, p.KeepRuns, p.Enabled, string(config), p.ID}
	}
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return fmt.Errorf("计划不存在")
	}
	return rows.Scan(&p.ID)
}
func (r *Store) ListQualitySchedules(ctx context.Context) ([]*quality.CodexQualitySchedule, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT id,name,interval_minutes,keep_runs,enabled,config,next_run_at,active_run_id,manual_requested_at FROM codex_quality_schedules ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*quality.CodexQualitySchedule{}
	for rows.Next() {
		p := &quality.CodexQualitySchedule{}
		var raw []byte
		if err = rows.Scan(&p.ID, &p.Name, &p.IntervalMinutes, &p.KeepRuns, &p.Enabled, &raw, &p.NextRunAt, &p.ActiveRunID, &p.ManualRequestedAt); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &p.Config); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (r *Store) SetQualityScheduleEnabled(ctx context.Context, id int64, enabled bool) error {
	result, err := r.sql.ExecContext(ctx, `WITH locked AS MATERIALIZED (SELECT id,active_run_id FROM codex_quality_schedules WHERE id=$1 FOR UPDATE), old AS (
		UPDATE codex_quality_runs SET status='interrupted',finished_at=NOW()
		WHERE id=(SELECT active_run_id FROM locked) AND status='running' AND NOT $2)
		UPDATE codex_quality_schedules SET enabled=$2,updated_at=NOW(),
		active_run_id=CASE WHEN $2 THEN active_run_id ELSE NULL END,
		lease_until=CASE WHEN $2 THEN lease_until ELSE NULL END,
		manual_requested_at=CASE WHEN $2 THEN manual_requested_at ELSE NULL END,
		next_run_at=CASE WHEN $2 AND NOT enabled THEN NOW()+make_interval(mins=>interval_minutes) ELSE next_run_at END
		WHERE id IN(SELECT id FROM locked)`, id, enabled)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return fmt.Errorf("计划不存在")
	}
	return err
}

// TriggerQualitySchedule 独立登记一次执行，不修改周期启停；重复点击在领取前合并。
func (r *Store) TriggerQualitySchedule(ctx context.Context, id int64) error {
	result, err := r.sql.ExecContext(ctx, `UPDATE codex_quality_schedules
		SET manual_requested_at=COALESCE(manual_requested_at,NOW()), updated_at=NOW()
		WHERE id=$1 AND (active_run_id IS NULL OR lease_until<NOW())`, id)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return quality.ErrQualityScheduleBusy
	}
	return nil
}
func (r *Store) ClaimQualitySchedule(ctx context.Context) (*quality.CodexQualityRun, error) {
	rows, err := r.sql.QueryContext(ctx, `WITH due AS MATERIALIZED (
		SELECT * FROM codex_quality_schedules WHERE (manual_requested_at IS NOT NULL OR (enabled AND next_run_at<=NOW()))
		AND (lease_until IS NULL OR lease_until<NOW()) ORDER BY manual_requested_at NULLS LAST,next_run_at,id LIMIT 1 FOR UPDATE SKIP LOCKED
	), old AS (
		UPDATE codex_quality_runs SET status='interrupted',finished_at=NOW()
		WHERE id IN(SELECT active_run_id FROM due) AND status='running'
	), created AS (
		INSERT INTO codex_quality_runs(schedule_id,schedule_name,config,trigger_source)
		SELECT id,name,config,CASE WHEN manual_requested_at IS NOT NULL THEN 'manual' ELSE 'schedule' END FROM due RETURNING *
	), leased AS (
		UPDATE codex_quality_schedules p SET active_run_id=c.id,lease_until=NOW()+INTERVAL '60 seconds',manual_requested_at=NULL
		FROM created c WHERE p.id=c.schedule_id RETURNING c.*
	) SELECT id,schedule_id,schedule_name,config,status,started_at,finished_at,trigger_source FROM leased`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, rows.Err()
	}
	return scanQualityRun(rows)
}

type qualityRowScanner interface{ Scan(...any) error }

// 删除只锁定目标计划，外键级联清理其轮次；与结果提交的计划共享锁串行化。
// @project-doc docs/operations/account_maintenance.md#codex_quality_schedules
func (r *Store) DeleteQualitySchedule(ctx context.Context, id int64) (bool, error) {
	result, err := r.sql.ExecContext(ctx, `DELETE FROM codex_quality_schedules WHERE id=$1`, id)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

func scanQualityRun(row qualityRowScanner) (*quality.CodexQualityRun, error) {
	run := &quality.CodexQualityRun{Counts: map[string]int{}}
	var raw []byte
	if err := row.Scan(&run.ID, &run.ScheduleID, &run.ScheduleName, &raw, &run.Status, &run.StartedAt, &run.FinishedAt, &run.TriggerSource); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &run.Config); err != nil {
		return nil, err
	}
	return run, nil
}
func (r *Store) RenewQualitySchedule(ctx context.Context, run *quality.CodexQualityRun) (bool, error) {
	result, err := r.sql.ExecContext(ctx, `UPDATE codex_quality_schedules SET lease_until=NOW()+INTERVAL '60 seconds'
	WHERE id=$1 AND active_run_id=$2 AND lease_until>NOW()
	AND (enabled OR EXISTS(SELECT 1 FROM codex_quality_runs WHERE id=$2 AND status='running' AND trigger_source='manual'))`, run.ScheduleID, run.ID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}
func (r *Store) SaveQualityRunResult(ctx context.Context, runID int64, result *quality.CodexQualityResult) error {
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	// 已在调度原子提交中保存的结果不覆盖；这里只补跳过/取消/存储异常。
	_, err = r.sql.ExecContext(ctx, `WITH plan_guard AS MATERIALIZED (
		SELECT p.id FROM codex_quality_schedules p JOIN codex_quality_runs r ON r.schedule_id=p.id
		WHERE r.id=$1 AND r.status='running' FOR SHARE OF p
	) INSERT INTO codex_quality_run_results(run_id,account_id,result)
	SELECT id,$2,$3::jsonb FROM codex_quality_runs WHERE id=$1 AND status='running'
	AND EXISTS(SELECT 1 FROM plan_guard)
	ON CONFLICT(run_id,account_id) DO NOTHING`, runID, result.AccountID, string(body))
	return err
}
func (r *Store) FinishQualityRun(ctx context.Context, run *quality.CodexQualityRun, status string) error {
	_, err := r.sql.ExecContext(ctx, `WITH owned AS MATERIALIZED (
		SELECT id,keep_runs FROM codex_quality_schedules WHERE id=$1 AND active_run_id=$2 FOR UPDATE
	), done AS (
		UPDATE codex_quality_runs SET status=$3,finished_at=NOW() WHERE id=$2 AND status='running'
		AND schedule_id IN(SELECT id FROM owned)
	), released AS (
		UPDATE codex_quality_schedules SET active_run_id=NULL,lease_until=NULL,
		next_run_at=NOW()+make_interval(mins=>interval_minutes) WHERE id IN(SELECT id FROM owned)
	), expired AS (
		SELECT id FROM codex_quality_runs WHERE schedule_id=$1 AND id<>$2 ORDER BY id DESC
		OFFSET GREATEST((SELECT keep_runs-1 FROM owned),0)
	) DELETE FROM codex_quality_runs WHERE id IN(SELECT id FROM expired)
	AND status<>'running' AND EXISTS(SELECT 1 FROM owned)`, run.ScheduleID, run.ID, status)
	return err
}
func (r *Store) ListQualityRuns(ctx context.Context, planID int64) ([]*quality.CodexQualityRun, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT id,schedule_id,schedule_name,config,status,started_at,finished_at,trigger_source
	FROM codex_quality_runs WHERE schedule_id=$1 ORDER BY id DESC LIMIT 100`, planID)
	if err != nil {
		return nil, err
	}
	out := []*quality.CodexQualityRun{}
	for rows.Next() {
		run, err := scanQualityRun(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		out = append(out, run)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	// 每个计划至多 100 轮，单次聚合填充分类统计，避免读取回答。
	counts, err := r.sql.QueryContext(ctx, `SELECT rr.run_id,rr.result->>'status',COUNT(*) FROM codex_quality_run_results rr
	JOIN codex_quality_runs r ON r.id=rr.run_id WHERE r.schedule_id=$1 GROUP BY rr.run_id,rr.result->>'status'`, planID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = counts.Close() }()
	byID := map[int64]*quality.CodexQualityRun{}
	for _, run := range out {
		byID[run.ID] = run
	}
	for counts.Next() {
		var id int64
		var status string
		var count int
		if err = counts.Scan(&id, &status, &count); err != nil {
			return nil, err
		}
		if run := byID[id]; run != nil {
			run.Counts[status] = count
		}
	}
	return out, counts.Err()
}
func (r *Store) GetQualityRun(ctx context.Context, id int64) (*quality.CodexQualityRun, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT id,schedule_id,schedule_name,config,status,started_at,finished_at,trigger_source FROM codex_quality_runs WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, fmt.Errorf("检测轮次不存在")
	}
	return scanQualityRun(rows)
}
func (r *Store) ListQualityRunResults(ctx context.Context, id int64, status string, page, size int) ([]*quality.CodexQualityResult, int, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT result,COUNT(*) OVER() FROM codex_quality_run_results
	WHERE run_id=$1 AND ($2='' OR result->>'status'=$2) ORDER BY account_id LIMIT $3 OFFSET $4`, id, status, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []*quality.CodexQualityResult{}
	total := 0
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw, &total); err != nil {
			return nil, 0, err
		}
		result := &quality.CodexQualityResult{}
		if err = json.Unmarshal(raw, result); err != nil {
			return nil, 0, err
		}
		out = append(out, result)
	}
	return out, total, rows.Err()
}
