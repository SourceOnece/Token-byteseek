package service

import (
	"context"
	"errors"
	"net/netip"
	"time"
)

// 只给手动采票使用的结构化诊断；来源是固定枚举，不返回 URL/原始网络错误。
type CodexTicketReferenceIP struct {
	CountryCode string `json:"country_code,omitempty"`
	Country     string `json:"country,omitempty"`
	Region      string `json:"region,omitempty"`
	City        string `json:"city,omitempty"`
	IP          string `json:"ip,omitempty"`
	Status      string `json:"status"`
	Source      string `json:"source,omitempty"`
	HTTPStatus  int    `json:"http_status,omitempty"`
}
type CodexTicketReferenceIPProber interface {
	ProbeTicketReferenceIP(context.Context, string) CodexTicketReferenceIP
}

func (s *CodexTicketService) collectReferenceIP(ctx context.Context, proxy codexTicketProxy, event *CodexTicketAttempt) {
	event.IPStatus = "not_attempted"
	if ctx.Err() != nil {
		event.IPStatus = "cancelled"
		return
	}
	if event.Diagnostic == nil || event.Diagnostic.HTTPStatus == 0 {
		return
	}
	if s.ipProber == nil {
		event.IPStatus = "unavailable"
		return
	}
	raw, err := s.cipher.Decrypt(proxy.Cipher)
	if err != nil || validateCodexHarvestProxy(raw) != nil {
		event.IPStatus = "proxy_config"
		return
	}
	ipCtx, stop := context.WithTimeout(ctx, 11*time.Second)
	defer stop()
	var info CodexTicketReferenceIP
	if prober, ok := s.ipProber.(CodexTicketReferenceIPProber); ok {
		info = prober.ProbeTicketReferenceIP(ipCtx, raw)
	} else {
		// 兼容旧实现/测试替身，失败仅给固定类别，绝不回显带密码的错误字符串。
		legacy, _, e := s.ipProber.ProbeProxy(ipCtx, raw)
		info.Status = "network"
		if e == nil && legacy != nil {
			info.IP = legacy.IP
			info.Status = "reference"
		}
		if errors.Is(e, context.DeadlineExceeded) {
			info.Status = "timeout"
		}
	}
	if ctx.Err() != nil {
		info.Status = "cancelled"
		info.IP = ""
	}
	event.IPStatus = info.Status
	event.IPSource = info.Source
	event.IPHTTPStatus = info.HTTPStatus
	now := time.Now().UTC()
	event.IPCheckedAt = &now
	if info.Status == "reference" {
		if ip, e := netip.ParseAddr(info.IP); e == nil {
			event.ReferenceIP = ip.String()
		} else {
			event.IPStatus = "invalid_response"
		}
	}
}
