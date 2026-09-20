package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// 固定/轮换使用地址列表，动态可选认证会话模板或每次请求取号API；所有地址仅保存密文。
type codexTicketProxyPolicy struct {
	ReuseSuccessfulIP bool               `json:"reuse_successful_ip,omitempty"`
	Mode              string             `json:"mode"`
	DynamicSource     string             `json:"dynamic_source,omitempty"`
	ExtractionCipher  string             `json:"extraction_cipher,omitempty"`
	ProxyProtocol     string             `json:"proxy_protocol,omitempty"`
	Proxies           []codexTicketProxy `json:"proxies,omitempty"`
	FixedProxyID      string             `json:"fixed_proxy_id,omitempty"`
}
type CodexTicketProxyPolicyView struct {
	ReuseSuccessfulIP    bool                   `json:"reuse_successful_ip"`
	Mode                 string                 `json:"mode"`
	DynamicSource        string                 `json:"dynamic_source"`
	ExtractionConfigured bool                   `json:"extraction_configured"`
	ProxyProtocol        string                 `json:"proxy_protocol"`
	Proxies              []CodexTicketProxyView `json:"proxies"`
	FixedProxyID         string                 `json:"fixed_proxy_id"`
}
type CodexTicketProxyPolicyUpdate struct {
	ReuseSuccessfulIP *bool                     `json:"reuse_successful_ip"`
	Mode              string                    `json:"mode"`
	DynamicSource     string                    `json:"dynamic_source"`
	ExtractionURL     *string                   `json:"extraction_url"`
	ProxyProtocol     string                    `json:"proxy_protocol"`
	Proxies           *[]CodexTicketProxyUpdate `json:"proxies"`
	FixedProxyID      *string                   `json:"fixed_proxy_id"`
}

type ticketProviderRejection struct {
	status  int
	retryAt *time.Time
}

func (e *ticketProviderRejection) Error() string { return "取号服务暂不可用" }

func (p *codexTicketProxyPolicy) configured() bool {
	if p == nil {
		return false
	}
	if p.Mode == "dynamic" && p.DynamicSource == "api" {
		return p.ExtractionCipher != ""
	}
	return len(p.Proxies) > 0
}
func (c *codexTicketConfig) applyProxyPolicy(p *codexTicketProxyPolicy) {
	c.ProxyPolicy = p
	c.Proxies = append([]codexTicketProxy(nil), p.Proxies...)
	c.SelectionMode = p.Mode
	c.FixedProxyID = p.FixedProxyID
	c.ProxyCipher = ""
	for _, v := range p.Proxies {
		if v.ID == p.FixedProxyID {
			c.ProxyCipher = v.Cipher
		}
	}
}
func ticketProxyPolicyView(c *codexTicketConfig) CodexTicketProxyPolicyView {
	p := c.ProxyPolicy
	if p == nil {
		p = &codexTicketProxyPolicy{Mode: c.mode(), DynamicSource: "template", ProxyProtocol: "http", Proxies: c.proxies(), FixedProxyID: c.FixedProxyID}
	}
	v := CodexTicketProxyPolicyView{ReuseSuccessfulIP: p.ReuseSuccessfulIP, Mode: p.Mode, DynamicSource: p.DynamicSource, ExtractionConfigured: p.ExtractionCipher != "", ProxyProtocol: p.ProxyProtocol, FixedProxyID: p.FixedProxyID, Proxies: []CodexTicketProxyView{}}
	if v.DynamicSource == "" {
		v.DynamicSource = "template"
	}
	if v.ProxyProtocol == "" {
		v.ProxyProtocol = "http"
	}
	for _, item := range p.Proxies {
		v.Proxies = append(v.Proxies, CodexTicketProxyView{ID: item.ID, Name: item.Name, Configured: item.Cipher != "" || item.ManagedID > 0, ManagedID: item.ManagedID})
	}
	return v
}
func validateTicketExtractionURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 4096 || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || strings.ContainsAny(raw, "\r\n\x00") {
		return errors.New("取号接口必须是有效的HTTPS地址")
	}
	if addr, e := netip.ParseAddr(u.Hostname()); e == nil && !ticketPublicIP(addr) {
		return errors.New("取号接口不能使用内网地址")
	}
	return nil
}
func (s *CodexTicketService) updateTicketProxyPolicy(c *codexTicketConfig, u *CodexTicketProxyPolicyUpdate) (*codexTicketProxyPolicy, error) {
	if u.Mode != "fixed" && u.Mode != "rotate" && u.Mode != "dynamic" {
		return nil, errors.New("无效的采集代理模式")
	}
	p := codexTicketProxyPolicy{Mode: u.Mode, DynamicSource: u.DynamicSource, ProxyProtocol: u.ProxyProtocol}
	if c.ProxyPolicy != nil {
		p.ExtractionCipher = c.ProxyPolicy.ExtractionCipher
		p.ReuseSuccessfulIP = c.ProxyPolicy.ReuseSuccessfulIP
	}
	// 旧调用方未提供时保留原值；固定模式不使用复用记录。
	if u.ReuseSuccessfulIP != nil {
		p.ReuseSuccessfulIP = *u.ReuseSuccessfulIP
	}
	if p.DynamicSource == "" {
		p.DynamicSource = "template"
	}
	if p.ProxyProtocol == "" {
		p.ProxyProtocol = "http"
	}
	if p.DynamicSource != "template" && p.DynamicSource != "api" {
		return nil, errors.New("无效的动态代理来源")
	}
	if p.ProxyProtocol != "http" && p.ProxyProtocol != "socks5h" {
		return nil, errors.New("取号代理协议仅支持HTTP或SOCKS5")
	}
	copy := *c
	copy.ProxyPolicy = nil
	mode := u.Mode
	if mode == "dynamic" {
		mode = "fixed"
	}
	if err := s.updateProxySettings(&copy, CodexTicketSettingsUpdate{Proxies: u.Proxies, SelectionMode: &mode, FixedProxyID: u.FixedProxyID}); err != nil {
		return nil, err
	}
	p.Proxies, p.FixedProxyID = copy.proxies(), copy.FixedProxyID
	if u.ExtractionURL != nil && strings.TrimSpace(*u.ExtractionURL) != "" {
		raw := strings.TrimSpace(*u.ExtractionURL)
		if err := validateTicketExtractionURL(raw); err != nil {
			return nil, err
		}
		if s.cipher == nil {
			return nil, errors.New("代理加密服务不可用")
		}
		encrypted, err := s.cipher.Encrypt(raw)
		if err != nil {
			return nil, errors.New("保存取号地址失败")
		}
		p.ExtractionCipher = encrypted
	}
	if !p.configured() {
		return nil, errors.New("请填写采集代理地址或取号接口")
	}
	return &p, nil
}

// 取号端点和返回代理只连接公网IP，解析后直接拨指定IP，避免再次DNS解析产生重绑定。
func ticketPublicIP(addr netip.Addr) bool {
	addr = addr.Unmap()
	return addr.IsGlobalUnicast() && !addr.IsPrivate() && !addr.IsLoopback() && !addr.IsLinkLocalUnicast()
}
func ticketResolvePublic(ctx context.Context, host string) (string, error) {
	if ip, err := netip.ParseAddr(host); err == nil {
		if !ticketPublicIP(ip) {
			return "", errors.New("private_address")
		}
		return ip.String(), nil
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return "", errors.New("dns_failed")
	}
	for _, ip := range ips {
		if !ticketPublicIP(ip) {
			return "", errors.New("private_address")
		}
	}
	return ips[0].String(), nil
}
func ticketProviderClient() *http.Client {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	t := &http.Transport{DisableKeepAlives: true, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 8 * time.Second, DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ip, err := ticketResolvePublic(ctx, host)
		if err != nil {
			return nil, err
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
	}}
	return &http.Client{Transport: t, Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// Mooproxy为JSON proxies数组四段式，也接受单条文本完整代理URL；只使用首条，不执行返回内容。
func parseTicketProviderResponse(body []byte, protocol string) (string, error) {
	text := strings.TrimSpace(string(body))
	if strings.HasPrefix(text, "{") {
		var payload struct {
			Proxies []string `json:"proxies"`
		}
		if json.Unmarshal(body, &payload) != nil || len(payload.Proxies) == 0 {
			return "", errors.New("取号响应格式无效")
		}
		text = strings.TrimSpace(payload.Proxies[0])
	} else {
		if line, _, ok := strings.Cut(text, "\n"); ok {
			text = strings.TrimSpace(line)
		}
	}
	if !strings.Contains(text, "://") {
		parts := strings.SplitN(text, ":", 4)
		if len(parts) != 2 && len(parts) != 4 {
			return "", errors.New("取号代理格式无效")
		}
		u := &url.URL{Scheme: protocol, Host: net.JoinHostPort(parts[0], parts[1])}
		if len(parts) == 4 {
			u.User = url.UserPassword(parts[2], parts[3])
		}
		text = u.String()
	}
	u, err := url.Parse(text)
	if err != nil || u.Scheme != "http" && u.Scheme != "socks5" && u.Scheme != "socks5h" {
		return "", errors.New("取号代理协议无效")
	}
	port, e := strconv.Atoi(u.Port())
	if e != nil || port < 1 || port > 65535 || validateCodexHarvestProxy(text) != nil {
		return "", errors.New("取号代理地址无效")
	}
	return text, nil
}
func (s *CodexTicketService) resolveTicketAttemptProxy(ctx context.Context, c *codexTicketConfig, p codexTicketProxy) (codexTicketProxy, error) {
	// 管理代理按ID引用，逐次读取最新地址和状态；删除/停用/过期即停止，不能回退直连。
	if p.ManagedID > 0 {
		if s.proxyRepo == nil || s.cipher == nil {
			return p, errors.New("管理代理服务不可用")
		}
		read, stop := context.WithTimeout(ctx, 2*time.Second)
		proxy, err := s.proxyRepo.GetByID(read, p.ManagedID)
		stop()
		if err != nil || proxy == nil || !proxy.IsActive() || proxy.IsExpired(time.Now()) {
			return p, errors.New("所选管理代理不可用")
		}
		if validateCodexHarvestProxy(expandTicketProxySession(proxy.URL())) != nil {
			return p, errors.New("管理代理协议无效")
		}
		p.Cipher, err = s.cipher.Encrypt(proxy.URL())
		if err != nil {
			return p, errors.New("代理配置不可读")
		}
	}
	if c.ProxyPolicy == nil || c.ProxyPolicy.Mode != "dynamic" || c.ProxyPolicy.DynamicSource != "api" {
		return s.ticketAttemptProxy(p)
	}
	raw, err := s.cipher.Decrypt(c.ProxyPolicy.ExtractionCipher)
	if err != nil || validateTicketExtractionURL(raw) != nil {
		return p, errors.New("取号配置不可用")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return p, errors.New("取号配置无效")
	}
	client := s.proxyProviderClient
	if client == nil {
		client = ticketProviderClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return p, errors.New("取号请求失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		until := codexTicketRetryNotBefore(resp.Header.Get("Retry-After"), resp.StatusCode, "")
		if until == nil && (resp.StatusCode == 401 || resp.StatusCode == 403) {
			at := time.Now().Add(5 * time.Minute)
			until = &at
		}
		return p, &ticketProviderRejection{status: resp.StatusCode, retryAt: until}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8193))
	if err != nil || len(body) > 8192 {
		return p, errors.New("取号响应不可读或过大")
	}
	proxy, err := parseTicketProviderResponse(body, c.ProxyPolicy.ProxyProtocol)
	if err != nil {
		return p, err
	}
	u, _ := url.Parse(proxy)
	ip, err := ticketResolvePublic(ctx, u.Hostname())
	if err != nil {
		return p, errors.New("取号代理不是可用的公网地址")
	}
	u.Host = net.JoinHostPort(ip, u.Port())
	p.Cipher, err = s.cipher.Encrypt(u.String())
	if err != nil {
		return p, errors.New("取号代理加密失败")
	}
	return p, nil
}
