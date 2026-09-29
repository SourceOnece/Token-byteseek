// 消费上限是可恢复的提供商状态；读取保持既有账期与软性重授权标记。
package provider

import (
	"context"
	"strings"
	"time"
)

// GrokReauthWriter 仅写入原有软性重新认证标记。
type GrokReauthWriter interface {
	UpdateExtra(context.Context, int64, map[string]any) error
}

// ClearGrokNeedsReauth 保留独立五秒写入预算和尽力失败语义。
func ClearGrokNeedsReauth(ctx context.Context, writer GrokReauthWriter, id int64) {
	if writer == nil || id <= 0 {
		return
	}
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	stateCtx, cancel := context.WithTimeout(base, 5*time.Second)
	defer cancel()
	_ = writer.UpdateExtra(stateCtx, id, map[string]any{
		"grok_needs_reauth":        false,
		"grok_needs_reauth_reason": "",
		"grok_needs_reauth_at":     "",
	})
}

const grokSpendingLimitProbeCooldown = 10 * time.Minute

func GrokSpendingLimitResetAt(provider *Record, now time.Time) time.Time {
	if provider != nil {
		if billing, err := ParseGrokBillingSnapshot(provider.Extra); err == nil && billing != nil {
			for _, raw := range []string{billing.PeriodEnd, billing.BillingPeriodEnd} {
				if resetAt, err := time.Parse(time.RFC3339, strings.TrimSpace(raw)); err == nil && resetAt.After(now) {
					return resetAt
				}
			}
		}
	}
	return now.Add(grokSpendingLimitProbeCooldown)
}

func GrokNeedsReauth(provider *Record) bool {
	if provider == nil {
		return false
	}
	if provider.Status == StatusError {
		msg := strings.ToLower(provider.ErrorMessage)
		if strings.Contains(msg, "spending limit") || strings.Contains(msg, "reauthorize") {
			return true
		}
	}
	if v, ok := provider.Extra["grok_needs_reauth"].(bool); ok && v {
		return true
	}
	if s, ok := provider.Extra["grok_needs_reauth"].(string); ok {
		return strings.EqualFold(s, "true") || s == "1"
	}
	return false
}
