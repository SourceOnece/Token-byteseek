-- 先保存 ByteSeek 的完整分组价格，转换至共享价格配置后再移除旧列。
-- 归档不是备份替代品，停机升级仍须保存整个数据库和应用数据。
CREATE TABLE IF NOT EXISTS byteseek_group_pricing_archive (
    group_id BIGINT PRIMARY KEY,
    original_group JSONB NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO byteseek_group_pricing_archive (group_id, original_group)
SELECT id, to_jsonb(g) FROM groups g ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS byteseek_channel_pricing_archive (
    channel_id BIGINT PRIMARY KEY,
    original_channel JSONB NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO byteseek_channel_pricing_archive (channel_id, original_channel)
SELECT id, to_jsonb(c) || jsonb_build_object('model_pricing', COALESCE((
    SELECT jsonb_agg(to_jsonb(p) || jsonb_build_object('intervals', COALESCE((
        SELECT jsonb_agg(to_jsonb(i) ORDER BY i.sort_order, i.id)
        FROM channel_pricing_intervals i WHERE i.pricing_id = p.id
    ), '[]'::jsonb)) ORDER BY p.id)
    FROM channel_model_pricing p WHERE p.channel_id = c.id
), '[]'::jsonb)) FROM channels c ON CONFLICT DO NOTHING;

-- 图片和视频统一使用模型价卡；旧单价与独立倍率直接清除，不转换已有 model_pricing。
-- 删列升级必须停止旧实例，历史账单和异步任务定价快照保持原样。
ALTER TABLE groups
    DROP COLUMN IF EXISTS image_price_1k,
    DROP COLUMN IF EXISTS image_price_2k,
    DROP COLUMN IF EXISTS image_price_4k,
    DROP COLUMN IF EXISTS video_price_480p,
    DROP COLUMN IF EXISTS video_price_720p,
    DROP COLUMN IF EXISTS video_price_1080p,
    DROP COLUMN IF EXISTS video_model_prices,
    DROP COLUMN IF EXISTS image_rate_independent,
    DROP COLUMN IF EXISTS image_rate_multiplier,
    DROP COLUMN IF EXISTS video_rate_independent,
    DROP COLUMN IF EXISTS video_rate_multiplier;
