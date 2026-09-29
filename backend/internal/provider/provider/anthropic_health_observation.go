package provider

import (
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/anthropic"
)

// QuotaWindowObservation 只投影供应商解析结果，健康规则由 provider 拥有。
func QuotaWindowObservation(value *anthropic.WindowLimit) *provider.QuotaWindowObservation {
	if value == nil {
		return nil
	}
	return &provider.QuotaWindowObservation{Window: value.Window, ResetAt: value.ResetAt, FiveHourReset: value.FiveHourReset, Reason: value.Reason}
}

// SessionWindowObservation 将原响应头投影为明确输入，不执行状态写入。
func SessionWindowObservation(headers http.Header) provider.SessionWindowObservation {
	return provider.SessionWindowObservation{Status: headers.Get("anthropic-ratelimit-unified-5h-status"), Reset: headers.Get("anthropic-ratelimit-unified-5h-reset"), Passive: anthropic.PassiveUsageFields(headers)}
}
