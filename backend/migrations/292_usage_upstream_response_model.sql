-- TokenFlux 285 重编号为 292。旧 response_model 留作兼容，不从请求模型猜测历史响应。
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS upstream_response_model VARCHAR(200);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS upstream_model_mismatch BOOLEAN;
