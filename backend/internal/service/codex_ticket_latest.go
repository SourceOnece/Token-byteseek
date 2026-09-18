package service

import (
	"context"
	"encoding/json"
	"net/netip"
	"time"
)

// 最新结果只用于展示，与真实票据有效性、后台重试观测隔离，避免展示反过来改变调度。
type CodexTicketLatest struct {
	Source       string                 `json:"source"`
	State        string                 `json:"state"`
	Reason       string                 `json:"reason,omitempty"`
	CheckedAt    time.Time              `json:"checked_at"`
	FinishedUS   int64                  `json:"finished_us"`
	Diagnostic   *CodexTicketDiagnostic `json:"diagnostic,omitempty"`
	ReferenceIP  string                 `json:"reference_ip,omitempty"`
	IPStatus     string                 `json:"ip_status,omitempty"`
	IPSource     string                 `json:"ip_source,omitempty"`
	IPHTTPStatus int                    `json:"ip_http_status,omitempty"`
}

func (s *CodexTicketService) recordLatest(ctx context.Context, cfg *codexTicketConfig, key, source string, e CodexTicketAttempt) {
	if s.cache == nil || key == "" || cfg == nil {
		return
	}
	current := s.enabledConfig()
	if current == nil || current.Generation != cfg.Generation {
		return
	}
	checked := e.FinishedAt
	if checked.IsZero() {
		checked = time.Now().UTC()
	}
	latest := CodexTicketLatest{Source: source, State: e.Status, Reason: e.Reason, CheckedAt: checked, FinishedUS: checked.UnixMicro(), Diagnostic: e.Diagnostic, ReferenceIP: e.ReferenceIP, IPStatus: e.IPStatus, IPSource: e.IPSource, IPHTTPStatus: e.IPHTTPStatus}
	safe := safeTicketLatest(latest)
	if safe == nil {
		return
	}
	raw, err := json.Marshal(safe)
	if err != nil {
		return
	}
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 150*time.Millisecond)
	defer cancel()
	_ = s.cache.SetLatest(writeCtx, key, string(raw), latest.FinishedUS, 24*time.Hour)
}

// 枚举与 IP 再次归一，缓存不能直接把任意文本/链接带入管理界面。
func safeTicketLatest(value CodexTicketLatest) *CodexTicketLatest {
	if value.CheckedAt.IsZero() {
		return nil
	}
	if value.Source != "manual" && value.Source != "auto" {
		return nil
	}
	switch value.State {
	case "ready", "missing", "failed", "cancelled", "skipped":
	default:
		return nil
	}
	switch value.Reason {
	case "network", "upstream", "invalid_ticket", "credential", "storage", "cancelled", "proxy_config", "ineligible", "account_changed", "concurrency_busy", "backoff":
	default:
		value.Reason = ""
	}
	value.Diagnostic = safeCodexTicketDiagnostic(value.Diagnostic)
	if ip, err := netip.ParseAddr(value.ReferenceIP); err == nil {
		value.ReferenceIP = ip.String()
	} else {
		value.ReferenceIP = ""
	}
	switch value.IPStatus {
	case "reference", "unavailable", "timeout", "network", "tls", "http_error", "invalid_response", "proxy_config", "cancelled", "not_attempted":
	default:
		value.IPStatus = ""
	}
	switch value.IPSource {
	case "chatgpt_trace", "ipify", "configured":
	default:
		value.IPSource = ""
	}
	if value.IPHTTPStatus < 100 || value.IPHTTPStatus > 599 {
		value.IPHTTPStatus = 0
	}
	return &value
}
