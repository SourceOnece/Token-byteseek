package provider

import (
	"context"
	"net/http"
	"time"
)

// ApplySparkRateLimit 将 Spark 独立配额窗口记录为模型级限流。
// Spark 的 x-codex-* 使用率和 reset 时间只代表 Spark 模型维度，不能写入提供商级
// RateLimitResetAt，否则同一 OAuth 提供商上的其他模型也会被错误停调。
func (s *HealthService) ApplySparkRateLimit(ctx context.Context, provider *Record, modelKey string, statusCode int, spark bool, observe func() (OpenAI429Disposition, *time.Time)) bool {
	if s == nil || provider == nil || s.providerRepo == nil || statusCode != http.StatusTooManyRequests || !provider.IsOpenAIOAuthLike() {
		return false
	}
	if !spark || !provider.ShouldHandleErrorCode(statusCode) {
		return false
	}

	if modelKey == "" {
		return false
	}
	now := s.options.Now()
	disposition, resetAt := observe()
	// Spark 只有明确耗尽 5h/7d 窗口时才能使用上游长 reset；普通瞬时 429
	// 即使携带全局 reset 头，也只能使用短时回避，避免错误冷却数天。
	if disposition != OpenAI429Quota5h && disposition != OpenAI429Quota7d {
		resetAt = nil
	}
	if resetAt == nil || !resetAt.After(now) {
		cooldown, ok := s.Fallback429Cooldown(ctx, provider)
		if !ok || cooldown <= 0 {
			cooldown = time.Duration(DefaultRateLimit429CooldownSeconds) * time.Second
		}
		reset := now.Add(cooldown)
		resetAt = &reset
	}
	if err := s.providerRepo.SetModelRateLimit(ctx, provider.ID, modelKey, *resetAt, CodexSparkRateLimitReason); err != nil {
		s.options.Warn("openai_codex_spark_model_rate_limit_set_failed", "provider_id", provider.ID, "model", modelKey, "error", err)
	}
	s.options.Info("openai_codex_spark_model_rate_limited", "provider_id", provider.ID, "model", modelKey, "reset_at", *resetAt)
	return true
}

// CodexSparkRateLimitReason 保留原 Spark 模型限流来源。
const CodexSparkRateLimitReason = "openai_codex_spark_rate_limit"
