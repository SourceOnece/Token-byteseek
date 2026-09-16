-- 独立新增调用白名单，不重命名或激活历史 models_list_config 展示配置。
ALTER TABLE groups ADD COLUMN IF NOT EXISTS model_allowlist JSONB NOT NULL DEFAULT '{"enabled":false}'::jsonb;

-- 规则更新纳入持久失效通知，复合 Key 也不能继续读取旧分组权限。
CREATE OR REPLACE FUNCTION enqueue_group_auth_cache_invalidation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    target_group_id BIGINT;
BEGIN
    target_group_id := OLD.id;
    IF TG_OP = 'UPDATE'
       AND OLD.status IS NOT DISTINCT FROM NEW.status
       AND OLD.is_exclusive IS NOT DISTINCT FROM NEW.is_exclusive
       AND OLD.allow_image_generation IS NOT DISTINCT FROM NEW.allow_image_generation
       AND OLD.model_allowlist IS NOT DISTINCT FROM NEW.model_allowlist
       AND OLD.deleted_at IS NOT DISTINCT FROM NEW.deleted_at THEN
        RETURN NEW;
    END IF;
    INSERT INTO auth_cache_invalidation_outbox (cache_key)
    SELECT encode(sha256(convert_to(k.key, 'UTF8')), 'hex')
    FROM api_keys AS k
    WHERE (k.group_id = target_group_id OR EXISTS (
        SELECT 1 FROM api_key_composite_groups AS binding
        WHERE binding.api_key_id = k.id AND binding.group_id = target_group_id
    )) AND k.deleted_at IS NULL AND k.key <> '';
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$;
