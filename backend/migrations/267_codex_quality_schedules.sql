-- 定时检测保存配置快照和每轮历史，原最近结果表保持兼容。
CREATE TABLE IF NOT EXISTS codex_quality_schedules (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    interval_minutes INTEGER NOT NULL CHECK (interval_minutes BETWEEN 1 AND 43200),
    keep_runs INTEGER NOT NULL DEFAULT 30 CHECK (keep_runs BETWEEN 1 AND 100),
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    config JSONB NOT NULL,
    next_run_at TIMESTAMPTZ NOT NULL,
    active_run_id BIGINT,
    lease_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS codex_quality_runs (
    id BIGSERIAL PRIMARY KEY,
    schedule_id BIGINT NOT NULL REFERENCES codex_quality_schedules(id) ON DELETE CASCADE,
    schedule_name TEXT NOT NULL,
    config JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_codex_quality_runs_schedule ON codex_quality_runs(schedule_id,id DESC);
CREATE TABLE IF NOT EXISTS codex_quality_run_results (
    run_id BIGINT NOT NULL REFERENCES codex_quality_runs(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL,
    result JSONB NOT NULL,
    PRIMARY KEY (run_id,account_id)
);
COMMENT ON TABLE codex_quality_run_results IS '管理员每轮检测结果；按计划保留轮数清理，不保存上游凭据';
