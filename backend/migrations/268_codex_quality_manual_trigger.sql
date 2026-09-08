-- 手动检测与周期启停独立；待执行请求持久化，重复点击不累计轮次。
ALTER TABLE codex_quality_schedules ADD COLUMN IF NOT EXISTS manual_requested_at TIMESTAMPTZ;
ALTER TABLE codex_quality_runs ADD COLUMN IF NOT EXISTS trigger_source TEXT NOT NULL DEFAULT 'schedule';
