package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

// 成功出口独立于票据有效期，持续获票则续期；闲置一天自动清理，不永久保留代理凭据。
const ticketProxyReuseTTL = 24 * time.Hour

var errTicketProxyReuseCache = errors.New("采集代理复用记录不可读")

type ticketProxyReuseRecord struct {
	Scope     string           `json:"scope"`
	Source    string           `json:"source"`
	Proxy     codexTicketProxy `json:"proxy"`
	ExpiresAt time.Time        `json:"expires_at"`
}

type ticketProxyReuseAttempt struct {
	key, source string
	reused      bool
}

func ticketProxyReuseEnabled(c *codexTicketConfig) bool {
	return c != nil && c.ProxyPolicy != nil && c.ProxyPolicy.ReuseSuccessfulIP && (c.mode() == "rotate" || c.mode() == "dynamic")
}

func ticketProxySourceHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// 管理代理必须逐次检查状态与地址。只有生成新代理时展开SID，复用时保留成功会话。
func (s *CodexTicketService) prepareTicketProxySource(ctx context.Context, c *codexTicketConfig, p codexTicketProxy) (codexTicketProxy, string, error) {
	if c.ProxyPolicy.DynamicSource == "api" && c.mode() == "dynamic" {
		return p, ticketProxySourceHash(c.ProxyPolicy.ExtractionCipher + ":" + c.ProxyPolicy.ProxyProtocol), nil
	}
	if p.ManagedID == 0 {
		return p, ticketProxySourceHash(p.Cipher), nil
	}
	if s.proxyRepo == nil || s.cipher == nil {
		return p, "", errors.New("管理代理服务不可用")
	}
	read, stop := context.WithTimeout(ctx, 2*time.Second)
	defer stop()
	proxy, err := s.proxyRepo.GetByID(read, p.ManagedID)
	if err != nil || proxy == nil || !proxy.IsActive() || proxy.IsExpired(time.Now()) {
		return p, "", errors.New("所选管理代理不可用")
	}
	raw := proxy.URL()
	if validateCodexHarvestProxy(expandTicketProxySession(raw)) != nil {
		return p, "", errors.New("管理代理协议无效")
	}
	p.Cipher, err = s.cipher.Encrypt(raw)
	if err != nil {
		return p, "", errors.New("代理配置不可读")
	}
	return p, ticketProxySourceHash(raw), nil
}

// @project-doc docs/interfaces/codex_ticket.md#ticket_contract
// 在原单模型采集租约内读取；记录与账号、模型、配置代次、Token和业务出口隔离。
func (s *CodexTicketService) resolveReusableTicketProxy(ctx context.Context, c *codexTicketConfig, key string, selected codexTicketProxy, observations ...codexTicketObservation) (codexTicketProxy, ticketProxyReuseAttempt, error) {
	if !ticketProxyReuseEnabled(c) {
		p, err := s.resolveTicketAttemptProxy(ctx, c, selected)
		return p, ticketProxyReuseAttempt{}, err
	}
	read, stop := context.WithTimeout(ctx, 200*time.Millisecond)
	encoded, err := s.cache.Get(read, "proxy-reuse:"+key)
	stop()
	if err != nil {
		return selected, ticketProxyReuseAttempt{}, errTicketProxyReuseCache
	}
	var saved ticketProxyReuseRecord
	if encoded != "" && len(encoded) <= 32768 {
		if raw, err := s.cipher.Decrypt(encoded); err == nil {
			if json.Unmarshal([]byte(raw), &saved) != nil {
				// 不能使用反序列化失败后残留的部分代理字段。
				saved = ticketProxyReuseRecord{}
			}
		}
	}
	// 清理写入偶发失败时，已有失败观测仍可阻止旧成功记录再次覆盖轮换选择。
	// 只采用成功记忆之后的观测，不让较早失败误删后来获得的同地址新会话。
	rejected := false
	if len(observations) > 0 {
		last := observations[0]
		if (last.State == "failed" || last.State == "missing") && last.Diagnostic != nil && last.Diagnostic.ProxyID == saved.Proxy.ID && last.CheckedAt.After(saved.ExpiresAt.Add(-ticketProxyReuseTTL)) && last.Diagnostic.RetryNotBefore == nil {
			switch last.Reason {
			case "invalid_ticket", "length_signal", "model_mismatch", "incomplete_response", "network", "proxy_provider", "timeout", "proxy_timeout", "transport_timeout":
				rejected = true
			}
		}
	}
	// 只接受仍在当前配置列表中的代理；即便缓存串键，也不能跨账号或模型使用。
	if !rejected && saved.Scope == key && saved.ExpiresAt.After(time.Now()) {
		for _, source := range c.proxies() {
			if source.ID != saved.Proxy.ID || source.ManagedID != saved.Proxy.ManagedID {
				continue
			}
			prepared, hash, sourceErr := s.prepareTicketProxySource(ctx, c, source)
			if sourceErr != nil {
				// 已停用的旧成功代理不再复用，保持原轮换失败处理而非偷偷直连。
				return source, ticketProxyReuseAttempt{key: key}, sourceErr
			}
			if hash == saved.Source {
				if raw, e := s.cipher.Decrypt(saved.Proxy.Cipher); e == nil && validateCodexHarvestProxy(raw) == nil {
					saved.Proxy.Name = source.Name
					return saved.Proxy, ticketProxyReuseAttempt{key: key, source: hash, reused: true}, nil
				}
			}
			// 管理代理地址变化时用当前配置重建，不再携带旧SID或旧密码。
			selected = prepared
			break
		}
	}
	// 不可复用的旧记录不能在下一次尝试中再次抢占轮换游标。
	if encoded != "" {
		write, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		err = s.cache.Set(write, "proxy-reuse:"+key, "", time.Second)
		cancel()
		if err != nil {
			return selected, ticketProxyReuseAttempt{}, errTicketProxyReuseCache
		}
	}
	prepared, hash, err := s.prepareTicketProxySource(ctx, c, selected)
	if err != nil {
		return selected, ticketProxyReuseAttempt{key: key}, err
	}
	managedID := prepared.ManagedID
	prepared.ManagedID = 0 // 已复核管理代理，避免再次读取后源指纹和实际地址来自不同快照。
	resolved, err := s.resolveTicketAttemptProxy(ctx, c, prepared)
	resolved.ManagedID = managedID
	return resolved, ticketProxyReuseAttempt{key: key, source: hash}, err
}

// 仅在最终存票成功后记忆；普通存储/凭据错误或限流不误判为IP不合格。
func (s *CodexTicketService) finishTicketProxyReuse(ctx context.Context, c *codexTicketConfig, attempt ticketProxyReuseAttempt, proxy codexTicketProxy, ready, retry bool, reason string, d *CodexTicketDiagnostic) {
	if attempt.key == "" || !s.ticketConfigCurrent(c) {
		return
	}
	write, stop := context.WithTimeout(context.WithoutCancel(ctx), 200*time.Millisecond)
	defer stop()
	if ready {
		value := ticketProxyReuseRecord{Scope: attempt.key, Source: attempt.source, Proxy: proxy, ExpiresAt: time.Now().Add(ticketProxyReuseTTL)}
		raw, err := json.Marshal(value)
		if err != nil {
			return
		}
		encoded, err := s.cipher.Encrypt(string(raw))
		if err == nil {
			_ = s.cache.Set(write, "proxy-reuse:"+attempt.key, encoded, ticketProxyReuseTTL)
		}
		return
	}
	// 清理在释放模型租约之前完成，不让下一轮拿到已确认失败的成功出口。
	failed := reason == "invalid_ticket" || reason == "length_signal" || reason == "model_mismatch" || reason == "incomplete_response" || reason == "proxy_provider" || ((reason == "network" || reason == "timeout" || reason == "proxy_timeout" || reason == "transport_timeout") && ctx.Err() == nil) || (reason == "upstream" && retry)
	if failed && (attempt.reused || reason == "proxy_provider") && (d == nil || d.RetryNotBefore == nil) {
		_ = s.cache.Set(write, "proxy-reuse:"+attempt.key, "", time.Second)
	}
}
