-- 价格设置统一归属价格配置；旧分组值不复制。升级前停止旧实例。
ALTER TABLE pricing_configs
 ADD COLUMN IF NOT EXISTS peak_rate_enabled boolean DEFAULT false NOT NULL,
 ADD COLUMN IF NOT EXISTS peak_start varchar(5) DEFAULT '' NOT NULL,
 ADD COLUMN IF NOT EXISTS peak_end varchar(5) DEFAULT '' NOT NULL,
 ADD COLUMN IF NOT EXISTS peak_rate_multiplier numeric(10,4) DEFAULT 1 NOT NULL,
 ADD COLUMN IF NOT EXISTS long_context_pricing_enabled boolean DEFAULT true NOT NULL,
 ADD COLUMN IF NOT EXISTS free_openai_fast boolean DEFAULT false NOT NULL,
 ADD COLUMN IF NOT EXISTS batch_image_discount_multiplier numeric(10,4) DEFAULT 0.5 NOT NULL,
 ADD COLUMN IF NOT EXISTS batch_image_hold_multiplier numeric(10,4) DEFAULT 0.6 NOT NULL,
 ADD COLUMN IF NOT EXISTS web_search_price_per_call numeric(20,8),
 ADD COLUMN IF NOT EXISTS search_price_per_1k numeric(20,8),
 ADD COLUMN IF NOT EXISTS audio_realtime_price_per_min numeric(20,8),
 ADD COLUMN IF NOT EXISTS audio_tts_price_per_million_chars numeric(20,8),
 ADD COLUMN IF NOT EXISTS audio_stt_price_per_hour numeric(20,8);

-- 旧分组字段保留作兼容归档；新价格配置是运行时唯一来源。
