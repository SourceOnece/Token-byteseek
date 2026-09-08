-- 管理员题目测试每账号只保留最近结果；临时租约防止多实例重复消耗额度。
CREATE TABLE IF NOT EXISTS codex_quality_tests (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    run_id TEXT NOT NULL,
    lease_until TIMESTAMPTZ NOT NULL,
    result JSONB,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE codex_quality_tests IS 'Codex 关键词题目测试的最近结果与并发租约，不保存账号凭据';
