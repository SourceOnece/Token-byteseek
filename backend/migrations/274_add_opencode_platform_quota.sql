-- sub2api 原迁移 238 的兼容适配：Go/Zen 共享平台身份，模式不拆分用户额度。
-- 保留本地 Qoder，不恢复已删除的 Composite 路由和上游独立监控表。
ALTER TABLE user_platform_quotas DROP CONSTRAINT IF EXISTS user_platform_quotas_platform_check;
ALTER TABLE user_platform_quotas ADD CONSTRAINT user_platform_quotas_platform_check
    CHECK (platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'qoder', 'grok',
                        'kimi', 'zhipu', 'deepseek', 'minimax', 'opencode_go'));
