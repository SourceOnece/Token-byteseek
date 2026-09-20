package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"time"
)

// 取消链路只携带固定来源，不把可能含凭据的网络原文写入日志。
type ticketStopCause string

func (c ticketStopCause) Error() string { return string(c) }

func ticketContextReason(ctx context.Context) string {
	if ctx == nil || ctx.Err() == nil {
		return ""
	}
	var cause ticketStopCause
	if errors.As(context.Cause(ctx), &cause) {
		return safeTicketReason(string(cause))
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "task_timeout"
	}
	return "client_disconnected"
}

func safeTicketReason(reason string) string {
	switch reason {
	case "network", "upstream", "invalid_ticket", "credential", "storage", "cancelled", "proxy_config", "proxy_provider", "cooldown", "ineligible", "account_changed", "concurrency_busy", "backoff", "business_proxy", "incomplete_response", "model_mismatch", "length_signal",
		"timeout", "transport_timeout", "proxy_timeout", "concurrency_config", "round_timeout", "task_timeout", "client_disconnected", "lease_lost", "config_unavailable", "service_stopped", "config_disabled", "account_missing", "unsupported_account", "account_inactive", "account_expired", "account_overloaded", "account_rate_limited", "account_cooldown", "model_unsupported", "attempt_limit", "unknown":
		return reason
	}
	return ""
}

func safeTicketPhase(phase string) string {
	switch phase {
	case "eligibility", "queue", "proxy", "harvest", "verify", "publish":
		return phase
	}
	return ""
}

// 当前资格与历史结果是两个时间点；只解释原资格检查，不放宽准入条件。
func ticketCollectionPause(account *Account) (string, *time.Time) {
	if account == nil {
		return "account_missing", nil
	}
	if !codexTicketAccount(account) {
		return "unsupported_account", nil
	}
	if !account.IsActive() {
		return "account_inactive", nil
	}
	now := time.Now()
	if account.AutoPauseOnExpired && account.ExpiresAt != nil && !now.Before(*account.ExpiresAt) {
		return "account_expired", nil
	}
	if account.OverloadUntil != nil && now.Before(*account.OverloadUntil) {
		return "account_overloaded", account.OverloadUntil
	}
	if account.RateLimitResetAt != nil && now.Before(*account.RateLimitResetAt) {
		return "account_rate_limited", account.RateLimitResetAt
	}
	if account.TempUnschedulableUntil != nil && now.Before(*account.TempUnschedulableUntil) {
		return "account_cooldown", account.TempUnschedulableUntil
	}
	copy := *account
	copy.Schedulable = true
	if !copy.IsSchedulable() {
		return "ineligible", nil
	}
	return "", nil
}

func (s *CodexTicketService) ticketConfigurationReason(cfg *codexTicketConfig) string {
	current := s.config.Load()
	if s.configUnavailable.Load() || current == nil || time.Since(current.loadedAt) > 10*time.Second {
		return "config_unavailable"
	}
	if !current.Enabled || (cfg != nil && !ticketConfigForAccount(current, cfg.accountID).Enabled) {
		return "config_disabled"
	}
	if !s.ticketConfigCurrent(cfg) {
		return "account_changed"
	}
	return ""
}

// 原始错误字符串不持久化，避免URL中的代理密码泄漏。
func ticketNetworkKind(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var timeout net.Error
	if errors.As(err, &timeout) && timeout.Timeout() {
		return "timeout"
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return "dns"
	}
	var cert *tls.CertificateVerificationError
	var authority x509.UnknownAuthorityError
	var host x509.HostnameError
	if errors.As(err, &cert) || errors.As(err, &authority) || errors.As(err, &host) {
		return "tls"
	}
	return "connection"
}

func ticketNetworkReason(ctx context.Context, err error, d *CodexTicketDiagnostic) string {
	d.NetworkKind = ticketNetworkKind(err)
	if reason := ticketContextReason(ctx); reason != "" {
		return reason
	}
	if d.NetworkKind == "timeout" {
		return "transport_timeout"
	}
	return "network"
}
