// EmailChallenges 拥有邮箱挑战与重置凭据；模板和发送由端口提供。
package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"time"
)

type EmailChallengeSender interface {
	SendVerifyCodeMessage(context.Context, string, string, string, string) error
	SendPasswordResetMessage(context.Context, string, string, string, string) error
}

// @project-doc docs/domains/identity_and_tenancy.md#email_challenges
type EmailChallenges struct {
	cache  EmailCache
	sender EmailChallengeSender
}

func NewEmailChallenges(cache EmailCache, sender EmailChallengeSender) *EmailChallenges {
	return &EmailChallenges{cache: cache, sender: sender}
}

// GenerateVerifyCode 生成6位数字验证码
func (s *EmailChallenges) GenerateVerifyCode() (string, error) {
	const digits = "0123456789"
	code := make([]byte, 6)
	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}
	return string(code), nil
}

// SendVerifyCode 发送验证码邮件
func (s *EmailChallenges) SendVerifyCode(ctx context.Context, email, siteName string, locale ...string) error {
	// 检查是否在冷却期内
	existing, err := s.cache.GetVerificationCode(ctx, email)
	if err == nil && existing != nil {
		if time.Since(existing.CreatedAt) < VerifyCodeCooldown {
			return ErrVerifyCodeTooFrequent
		}
	}

	// 生成验证码
	code, err := s.GenerateVerifyCode()
	if err != nil {
		return fmt.Errorf("generate code: %w", err)
	}

	// 保存验证码到 Redis
	data := &VerificationCodeData{
		Code:      code,
		Attempts:  0,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(VerifyCodeTTL),
	}
	if err := s.cache.SetVerificationCode(ctx, email, data, VerifyCodeTTL); err != nil {
		return fmt.Errorf("save verify code: %w", err)
	}

	return s.sender.SendVerifyCodeMessage(ctx, email, siteName, code, FirstEmailLocale(locale))
}

// VerifyCode 验证验证码
func (s *EmailChallenges) VerifyCode(ctx context.Context, email, code string) error {
	return verifyCodeWithAttempts(ctx, email, code, s.cache.GetVerificationCode, s.cache.IncrVerificationCodeAttempts, func() {
		if err := s.cache.DeleteVerificationCode(ctx, email); err != nil {
			slog.Error("failed to delete verification code after success", "email", email, "error", err)
		}
	})
}

// verifyCodeWithAttempts 在比较前预占次数，注册与通知邮箱共用同一上限语义。
func verifyCodeWithAttempts(ctx context.Context, email, code string,
	get func(context.Context, string) (*VerificationCodeData, error),
	incr func(context.Context, string) (int, error), onSuccess func(),
) error {
	data, err := get(ctx, email)
	if err != nil || data == nil {
		return ErrInvalidVerifyCode
	}

	// 检查是否已达到最大尝试次数
	if data.Attempts >= MaxVerifyCodeAttempts {
		return ErrVerifyCodeMaxAttempts
	}

	attempts, err := incr(ctx, email)
	if err != nil {
		return ErrInvalidVerifyCode
	}
	if attempts > MaxVerifyCodeAttempts {
		return ErrVerifyCodeMaxAttempts
	}
	// 保持常量时间比较，失败不会重新写验证码或延长 TTL。
	if subtle.ConstantTimeCompare([]byte(data.Code), []byte(code)) != 1 {
		if attempts >= MaxVerifyCodeAttempts {
			return ErrVerifyCodeMaxAttempts
		}
		return ErrInvalidVerifyCode
	}

	if onSuccess != nil {
		onSuccess()
	}
	return nil
}

// GeneratePasswordResetToken generates a secure 32-byte random token (64 hex characters)
func (s *EmailChallenges) GeneratePasswordResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// SendPasswordResetEmail sends a password reset email with a reset link
func (s *EmailChallenges) SendPasswordResetEmail(ctx context.Context, email, siteName, resetURL string, locale ...string) error {
	// 只保存摘要，每次允许重发时生成新链接，旧链接随之失效。
	token, err := s.GeneratePasswordResetToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	data := &PasswordResetTokenData{Token: hashPasswordResetToken(token), CreatedAt: time.Now()}
	if err := s.cache.SetPasswordResetToken(ctx, email, data, PasswordResetTokenTTL); err != nil {
		return fmt.Errorf("save reset token: %w", err)
	}

	// Build full reset URL with URL-encoded token and email
	fullResetURL := fmt.Sprintf("%s?email=%s&token=%s", resetURL, url.QueryEscape(email), url.QueryEscape(token))

	return s.sender.SendPasswordResetMessage(ctx, email, siteName, fullResetURL, FirstEmailLocale(locale))
}

// SendPasswordResetEmailWithCooldown sends password reset email with cooldown check (called by queue worker)
// This method wraps SendPasswordResetEmail with email cooldown to prevent email bombing
func (s *EmailChallenges) SendPasswordResetEmailWithCooldown(ctx context.Context, email, siteName, resetURL string, locale ...string) error {
	// Check email cooldown to prevent email bombing
	if s.cache.IsPasswordResetEmailInCooldown(ctx, email) {
		slog.Info("password reset email skipped due to cooldown", "email", email)
		return nil // Silent success to prevent revealing cooldown to attackers
	}

	// Send email using core method
	if err := s.SendPasswordResetEmail(ctx, email, siteName, resetURL, FirstEmailLocale(locale)); err != nil {
		return err
	}

	// Set cooldown marker (Redis TTL handles expiration)
	if err := s.cache.SetPasswordResetEmailCooldown(ctx, email, PasswordResetEmailCooldown); err != nil {
		slog.Error("failed to set password reset cooldown", "email", email, "error", err)
	}

	return nil
}

// VerifyPasswordResetToken verifies the password reset token without consuming it
func (s *EmailChallenges) VerifyPasswordResetToken(ctx context.Context, email, token string) error {
	data, err := s.cache.GetPasswordResetToken(ctx, email)
	if err != nil || data == nil || token == "" {
		return ErrInvalidResetToken
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(data.Token), []byte(hashPasswordResetToken(token))) != 1 {
		return ErrInvalidResetToken
	}

	return nil
}

// ConsumePasswordResetToken verifies and deletes the token (one-time use)
func (s *EmailChallenges) ConsumePasswordResetToken(ctx context.Context, email, token string) error {
	if token == "" {
		return ErrInvalidResetToken
	}
	consumed, err := s.cache.ConsumePasswordResetToken(ctx, email, hashPasswordResetToken(token))
	if err != nil || !consumed {
		return ErrInvalidResetToken
	}
	return nil
}

// hashPasswordResetToken 仅用于缓存摘要，不把摘要发给用户当作凭据。
func hashPasswordResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
