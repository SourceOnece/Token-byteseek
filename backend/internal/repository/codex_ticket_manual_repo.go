package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/TokenFlux/TokenRouter/internal/service"
)

// 复用账号仓储的数据库连接，但历史只写独立表，不修改账号或调度 outbox。
func (r *accountRepository) CreateTicketRun(ctx context.Context, run *service.CodexTicketManualRun) error {
	config, err := json.Marshal(run.Config)
	if err != nil {
		return err
	}
	_, err = r.sql.ExecContext(ctx, `INSERT INTO codex_ticket_manual_runs(id,status,config,total,started_at) VALUES($1,'running',$2,$3,$4)`, run.ID, string(config), run.Total, run.StartedAt)
	return err
}
func (r *accountRepository) AppendTicketEvent(ctx context.Context, id string, e *service.CodexTicketAttempt) error {
	raw, err := json.Marshal(e)
	if err != nil {
		return err
	}
	rows, err := r.sql.QueryContext(ctx, `INSERT INTO codex_ticket_manual_events(run_id,kind,account_id,model,status,result)
	SELECT id,$2,$3,$4,$5,$6 FROM codex_ticket_manual_runs WHERE id=$1 AND status='running' RETURNING id`, id, e.Kind, e.AccountID, e.Model, e.Status, string(raw))
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return errors.New("采集批次已结束")
	}
	return rows.Scan(&e.ID)
}
func (r *accountRepository) HeartbeatTicketRun(ctx context.Context, id string) error {
	result, err := r.sql.ExecContext(ctx, `UPDATE codex_ticket_manual_runs SET heartbeat_at=NOW() WHERE id=$1 AND status='running'`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return service.ErrTicketHistoryNotFound
	}
	return nil
}
func (r *accountRepository) FinishTicketRun(ctx context.Context, id, status string, counts map[string]int) error {
	raw, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	_, err = r.sql.ExecContext(ctx, `UPDATE codex_ticket_manual_runs SET status=$2,counts=$3,finished_at=NOW(),heartbeat_at=NOW() WHERE id=$1 AND status='running'`, id, status, string(raw))
	return err
}

// 失联批次只读呈现为 interrupted，不在 GET 中恢复或重新执行采集。
const ticketRunColumns = `id,CASE WHEN status='running' AND heartbeat_at<NOW()-INTERVAL '60 seconds' THEN 'interrupted' ELSE status END,config,total,started_at,finished_at,
COALESCE((SELECT jsonb_object_agg(status,n) FROM (SELECT e.status,count(*) n FROM codex_ticket_manual_events e WHERE e.run_id=r.id AND e.kind='result' GROUP BY e.status) c),'{}'::jsonb)`

func (r *accountRepository) ListTicketRuns(ctx context.Context, page int) ([]service.CodexTicketManualRun, error) {
	if page < 1 {
		page = 1
	}
	rows, err := r.sql.QueryContext(ctx, `SELECT `+ticketRunColumns+` FROM codex_ticket_manual_runs r ORDER BY started_at DESC LIMIT 20 OFFSET $1`, (page-1)*20)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.CodexTicketManualRun{}
	for rows.Next() {
		var run service.CodexTicketManualRun
		var config, counts []byte
		if err = rows.Scan(&run.ID, &run.Status, &config, &run.Total, &run.StartedAt, &run.FinishedAt, &counts); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(config, &run.Config); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(counts, &run.Counts); err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}
func (r *accountRepository) GetTicketRun(ctx context.Context, id string) (*service.CodexTicketManualRun, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT `+ticketRunColumns+` FROM codex_ticket_manual_runs r WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, errors.New("采集批次不存在")
	}
	var run service.CodexTicketManualRun
	var config, counts []byte
	if err = rows.Scan(&run.ID, &run.Status, &config, &run.Total, &run.StartedAt, &run.FinishedAt, &counts); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(config, &run.Config); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(counts, &run.Counts); err != nil {
		return nil, err
	}
	return &run, nil
}
func (r *accountRepository) ListTicketEvents(ctx context.Context, id, kind, status string, accountID int64, model string, page int) ([]service.CodexTicketAttempt, int, error) {
	if page < 1 {
		page = 1
	}
	where := ` FROM codex_ticket_manual_events WHERE run_id=$1 AND kind=$2 AND ($3='' OR status=$3) AND ($4::bigint=0 OR account_id=$4) AND ($5='' OR model=$5)`
	args := []any{id, kind, status, accountID, model}
	rows, err := r.sql.QueryContext(ctx, `SELECT count(*)`+where, args...)
	if err != nil {
		return nil, 0, err
	}
	var total int
	if rows.Next() {
		err = rows.Scan(&total)
	}
	rows.Close()
	if err != nil {
		return nil, 0, err
	}
	rows, err = r.sql.QueryContext(ctx, `SELECT id,result`+where+` ORDER BY id DESC LIMIT 50 OFFSET $6`, append(args, (page-1)*50)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []service.CodexTicketAttempt{}
	for rows.Next() {
		var e service.CodexTicketAttempt
		var raw []byte
		var eventID int64
		if err = rows.Scan(&eventID, &raw); err != nil {
			return nil, 0, err
		}
		if err = json.Unmarshal(raw, &e); err != nil {
			return nil, 0, err
		}
		e.ID = eventID
		out = append(out, e)
	}
	return out, total, rows.Err()
}

// 父批次加行锁，删除与心跳/终态串行；仅已结束或超过60秒失联的历史可清理。
// @project-doc docs/interfaces/codex_ticket.md#manual_collection
func (r *accountRepository) DeleteTicketHistory(ctx context.Context, runID string, eventID int64) (int64, error) {
	if runID == "" {
		rows, err := r.sql.QueryContext(ctx, `WITH eligible AS MATERIALIZED (
		 SELECT id FROM codex_ticket_manual_runs WHERE status<>'running' OR heartbeat_at<NOW()-INTERVAL '60 seconds' FOR UPDATE
		), removed AS (DELETE FROM codex_ticket_manual_runs WHERE id IN(SELECT id FROM eligible) RETURNING id)
		SELECT count(*) FROM removed`)
		if err != nil {
			return 0, err
		}
		defer rows.Close()
		var n int64
		if !rows.Next() {
			return 0, rows.Err()
		}
		err = rows.Scan(&n)
		return n, err
	}
	var query string
	args := []any{runID}
	guard := `WITH locked AS MATERIALIZED (SELECT id,(status='running' AND heartbeat_at>=NOW()-INTERVAL '60 seconds') AS active FROM codex_ticket_manual_runs WHERE id=$1 FOR UPDATE), removed AS (`
	if eventID == 0 {
		query = guard + `DELETE FROM codex_ticket_manual_runs WHERE id IN(SELECT id FROM locked WHERE NOT active) RETURNING id)
		SELECT EXISTS(SELECT 1 FROM locked),COALESCE((SELECT active FROM locked),false),(SELECT count(*) FROM removed)`
	} else {
		query = guard + `DELETE FROM codex_ticket_manual_events WHERE run_id IN(SELECT id FROM locked WHERE NOT active) AND id=$2 RETURNING id)
		SELECT EXISTS(SELECT 1 FROM locked),COALESCE((SELECT active FROM locked),false),(SELECT count(*) FROM removed)`
		args = append(args, eventID)
	}
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var exists, active bool
	var n int64
	if !rows.Next() {
		return 0, errors.New("删除采集日志失败")
	}
	if err = rows.Scan(&exists, &active, &n); err != nil {
		return 0, err
	}
	if active {
		return 0, service.ErrTicketHistoryActive
	}
	if !exists || n == 0 {
		return 0, service.ErrTicketHistoryNotFound
	}
	return n, nil
}
