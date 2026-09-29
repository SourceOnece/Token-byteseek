-- 系列默认映射下线，只清理对应 JSON 字段，保留精确模型覆盖和其它分组配置。
-- ByteSeek 保留原系列映射，避免未迁移的管理员规则丢失；新路由规则优先读取 routing_policy。

COMMENT ON COLUMN groups.messages_dispatch_model_config IS 'Messages 精确模型映射配置';
