-- 手动采票历史独立保存，禁止把凭据/票据或代理密码写入此表；不改变账号调度。
CREATE TABLE IF NOT EXISTS codex_ticket_manual_runs (
    id UUID PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'running',
    config JSONB NOT NULL,
    total INTEGER NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    heartbeat_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    counts JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS idx_codex_ticket_manual_runs_started ON codex_ticket_manual_runs(started_at DESC);
CREATE TABLE IF NOT EXISTS codex_ticket_manual_events (
    id BIGSERIAL PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES codex_ticket_manual_runs(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    account_id BIGINT NOT NULL,
    model TEXT NOT NULL,
    status TEXT NOT NULL,
    result JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_codex_ticket_manual_events_run ON codex_ticket_manual_events(run_id, id);
CREATE INDEX IF NOT EXISTS idx_codex_ticket_manual_results ON codex_ticket_manual_events(run_id, kind, status, id);
