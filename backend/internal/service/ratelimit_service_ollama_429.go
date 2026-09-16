package service

// Ollama 429 先维持不缩短的临时冷却，再异步查询真实用量重置时间。
// 回写校验账号身份及限流代次，凭据更新、管理员清除和新 429 都不会被旧结果覆盖。

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

const ollamaCloudUsageProbeWritebackTimeout = 10 * time.Second

type ollamaCloudUsageProbeScheduler interface {
	ScheduleOllamaCloudUsageRateLimitProbe(accountID int64, onExhausted OllamaCloudUsageRateLimitProbeCallback) bool
}

type ollamaCloudUsageRateLimitExtender interface {
	SetRateLimitedIfLater(ctx context.Context, id int64, resetAt time.Time) error
}

type ollamaCloudUsageRateLimitSetterIfGeneration interface {
	SetRateLimitedIfUnchanged(ctx context.Context, id int64, expectedUpdatedAt time.Time, expectedLimitedAt, expectedResetAt *time.Time, newResetAt time.Time) (bool, error)
}

func (s *RateLimitService) SetOllamaCloudUsageProbeScheduler(scheduler ollamaCloudUsageProbeScheduler) {
	s.ollamaCloudUsageProbe = scheduler
}

func (s *RateLimitService) handleOllamaCloudUsage429(ctx context.Context, account *Account, headers http.Header) {
	if s == nil || account == nil || account.ID <= 0 || s.accountRepo == nil {
		return
	}

	var shortReset time.Time
	now := time.Now()
	if d := ollamaCloudUsageRetryAfter(headers, now); d > 0 {
		shortReset = now.Add(d)
	} else if cooldown, enabled := s.get429FallbackCooldown(ctx, account); enabled {
		shortReset = now.Add(cooldown)
	} else {
		slog.Info("rate_limit_ollama_429_fallback_ignored", "account_id", account.ID, "platform", account.Platform)
	}

	if !shortReset.IsZero() {
		s.applyOllamaCloudUsageImmediateCooldown(ctx, account, shortReset)
	}

	authoritative, err := s.accountRepo.GetByID(ctx, account.ID)
	if err != nil || authoritative == nil {
		slog.Warn("ollama_cloud_usage_authoritative_load_failed", "account_id", account.ID, "error", err)
		return
	}
	if authoritative.RateLimitResetAt != nil && authoritative.RateLimitResetAt.After(now) {
		s.notifyAccountSchedulingBlocked(authoritative, *authoritative.RateLimitResetAt, "ollama_429")
	}

	if s.ollamaCloudUsageProbe == nil {
		return
	}
	s.scheduleOllamaCloudUsageProbe(authoritative)
}

func (s *RateLimitService) applyOllamaCloudUsageImmediateCooldown(ctx context.Context, account *Account, shortReset time.Time) {
	if extender, ok := s.accountRepo.(ollamaCloudUsageRateLimitExtender); ok {
		if err := extender.SetRateLimitedIfLater(ctx, account.ID, shortReset); err != nil {
			slog.Warn("rate_limit_ollama_429_iflater_failed", "account_id", account.ID, "error", err)
		}
		return
	}
	reset := shortReset
	if account.RateLimitResetAt != nil && account.RateLimitResetAt.After(reset) {
		reset = *account.RateLimitResetAt
	}
	if err := s.accountRepo.SetRateLimited(ctx, account.ID, reset); err != nil {
		slog.Warn("rate_limit_set_failed", "account_id", account.ID, "error", err)
	}
}

func (s *RateLimitService) scheduleOllamaCloudUsageProbe(account *Account) {
	if s == nil || account == nil || s.ollamaCloudUsageProbe == nil {
		return
	}
	if !IsOllamaCloudUsageAccount(account) {
		return
	}
	fingerprint, valid := ollamaCloudUsageGroupFingerprint(account)
	if !valid {
		return
	}
	expectedLimitedAt := cloneTimePtr(account.RateLimitedAt)
	expectedResetAt := cloneTimePtr(account.RateLimitResetAt)
	accepted := s.ollamaCloudUsageProbe.ScheduleOllamaCloudUsageRateLimitProbe(
		account.ID,
		func(accountID int64, resetAt time.Time) {
			s.applyOllamaCloudUsageProbeReset(accountID, fingerprint, expectedLimitedAt, expectedResetAt, resetAt)
		},
	)
	if !accepted {
		slog.Debug("ollama_cloud_usage_probe_schedule_rejected", "account_id", account.ID)
	}
}

func (s *RateLimitService) applyOllamaCloudUsageProbeReset(
	accountID int64,
	expectedFingerprint string,
	expectedLimitedAt, expectedResetAt *time.Time,
	resetAt time.Time,
) {
	now := time.Now()
	if s == nil || accountID <= 0 || s.accountRepo == nil || !resetAt.After(now) {
		return
	}
	if expectedResetAt != nil && !resetAt.After(*expectedResetAt) {
		return
	}

	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), ollamaCloudUsageProbeWritebackTimeout)
	defer cancel()

	account, err := s.accountRepo.GetByID(bgCtx, accountID)
	if err != nil || account == nil {
		slog.Warn("ollama_cloud_usage_probe_reset_load_failed", "account_id", accountID, "error", err)
		return
	}
	if !IsOllamaCloudUsageAccount(account) {
		return
	}
	if !account.IsActive() || !account.Schedulable {
		return
	}
	currentFingerprint, valid := ollamaCloudUsageGroupFingerprint(account)
	if !valid || currentFingerprint != expectedFingerprint {
		return
	}

	setter, ok := s.accountRepo.(ollamaCloudUsageRateLimitSetterIfGeneration)
	if !ok {
		return
	}
	updated, err := setter.SetRateLimitedIfUnchanged(bgCtx, accountID, account.UpdatedAt, expectedLimitedAt, expectedResetAt, resetAt)
	if err != nil {
		slog.Warn("rate_limit_set_failed", "account_id", accountID, "error", err)
		return
	}
	if !updated {
		slog.Debug("ollama_cloud_usage_probe_reset_skipped_stale", "account_id", accountID)
		return
	}

	s.notifyAccountSchedulingBlocked(account, resetAt, "ollama_cloud_usage_429_probe")
	slog.Info("ollama_cloud_account_rate_limited_probe",
		"account_id", accountID,
		"reset_at", resetAt.UTC(),
		"reset_in", time.Until(resetAt).Truncate(time.Second),
	)
}
