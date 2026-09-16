-- sub2api 原迁移 237 的兼容适配，只扩展本 fork 实际拥有的额度表。
-- 不恢复已删除的 Composite 平台或另一套监控表，不回填已有用户额度。
ALTER TABLE user_platform_quotas DROP CONSTRAINT IF EXISTS user_platform_quotas_platform_check;
ALTER TABLE user_platform_quotas ADD CONSTRAINT user_platform_quotas_platform_check
    CHECK (platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'qoder', 'grok',
                        'kimi', 'zhipu', 'deepseek', 'minimax'));
