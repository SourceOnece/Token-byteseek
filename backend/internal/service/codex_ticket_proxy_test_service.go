package service

import (
	"context"
	"errors"
	"time"
)

type CodexTicketProxyTestRequest struct {
	AccountID int64                         `json:"account_id"`
	Policy    *CodexTicketProxyPolicyUpdate `json:"policy"`
	ProxyID   string                        `json:"proxy_id"`
	Confirmed bool                          `json:"confirmed"`
}
type CodexTicketProxyTestResult struct {
	CodexTicketReferenceIP
	DurationMS int64 `json:"duration_ms"`
}

// 显式测试只取一条代理并查出口，不调用Codex，不保存草稿，不自动切代理掩盖失败。
func (s *CodexTicketService) TestTicketProxy(ctx context.Context, input CodexTicketProxyTestRequest) (CodexTicketProxyTestResult, error) {
	if input.AccountID < 0 {
		return CodexTicketProxyTestResult{}, errors.New("账号ID无效")
	}
	if !input.Confirmed {
		return CodexTicketProxyTestResult{}, errors.New("请确认代理测试会使用服务商流量")
	}
	cfg, err := s.readConfig(ctx)
	if err != nil {
		return CodexTicketProxyTestResult{}, err
	}
	if input.AccountID > 0 {
		if _, err = s.AccountSettings(ctx, input.AccountID); err != nil {
			return CodexTicketProxyTestResult{}, err
		}
		cfg = ticketConfigForAccount(cfg, input.AccountID)
	}
	if input.Policy != nil && input.Policy.Mode != "inherit" {
		p, e := s.updateTicketProxyPolicy(cfg, input.Policy)
		if e != nil {
			return CodexTicketProxyTestResult{}, e
		}
		copy := *cfg
		copy.applyProxyPolicy(p)
		cfg = &copy
	}
	if input.Policy != nil && input.Policy.Mode == "inherit" {
		cfg, err = s.readConfig(ctx)
		if err != nil {
			return CodexTicketProxyTestResult{}, err
		}
	}
	proxy, ok := selectCodexTicketProxy(cfg, "", false)
	if input.ProxyID != "" {
		ok = false
		for _, p := range cfg.proxies() {
			if p.ID == input.ProxyID {
				proxy, ok = p, true
				break
			}
		}
	}
	if !ok {
		return CodexTicketProxyTestResult{}, errors.New("没有可测试的采集代理")
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	start := time.Now()
	proxy, err = s.resolveTicketAttemptProxy(ctx, cfg, proxy)
	if err != nil {
		return CodexTicketProxyTestResult{CodexTicketReferenceIP: CodexTicketReferenceIP{Status: "proxy_provider"}, DurationMS: time.Since(start).Milliseconds()}, nil
	}
	raw, err := s.cipher.Decrypt(proxy.Cipher)
	if err != nil {
		return CodexTicketProxyTestResult{}, errors.New("代理配置不可读")
	}
	prober, ok := s.ipProber.(CodexTicketReferenceIPProber)
	if !ok {
		return CodexTicketProxyTestResult{}, errors.New("代理测试服务不可用")
	}
	info := prober.ProbeTicketReferenceIP(ctx, raw)
	return CodexTicketProxyTestResult{CodexTicketReferenceIP: info, DurationMS: time.Since(start).Milliseconds()}, nil
}
