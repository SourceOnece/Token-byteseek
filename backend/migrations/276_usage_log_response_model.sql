-- 管理员独立记录上游声明的响应模型；历史记录保持NULL，不影响请求模型或计费。
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS response_model VARCHAR(200);
