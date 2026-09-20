package service

import (
	"errors"
	"github.com/google/uuid"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// 代理列表只保存密文；名称与 ID 用于只读诊断，地址永不回显。
type codexTicketProxy struct {
	ManagedID int64  `json:"managed_proxy_id,omitempty"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Cipher    string `json:"cipher"`
}
type CodexTicketProxyView struct {
	ManagedID  int64  `json:"managed_proxy_id,omitempty"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Configured bool   `json:"configured"`
}
type CodexTicketProxyUpdate struct {
	ManagedID int64  `json:"managed_proxy_id,omitempty"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"harvest_proxy_url"`
}

func (c *codexTicketConfig) proxies() []codexTicketProxy {
	if c.ProxyPolicy != nil && c.ProxyPolicy.Mode == "dynamic" && c.ProxyPolicy.DynamicSource == "api" {
		return []codexTicketProxy{{ID: "provider", Name: "API"}}
	}
	if len(c.Proxies) > 0 {
		return c.Proxies
	}
	if c.ProxyCipher != "" {
		return []codexTicketProxy{{ID: "legacy", Name: "Proxy 1", Cipher: c.ProxyCipher}}
	}
	return nil
}

func (c *codexTicketConfig) hasCollectionProxy() bool {
	if c.ProxyPolicy != nil && c.ProxyPolicy.configured() {
		return true
	}
	if len(c.proxies()) > 0 {
		return true
	}
	for _, account := range c.Accounts {
		if account.Mode != "off" && account.ProxyPolicy != nil && account.ProxyPolicy.configured() {
			return true
		}
		if account.Mode != "off" && (account.ProxyCipher != "" || account.ProxyPolicy != nil && account.ProxyPolicy.configured()) {
			return true
		}
	}
	return false
}
func (c *codexTicketConfig) mode() string {
	if c.SelectionMode == "dynamic" {
		return "dynamic"
	}
	if c.SelectionMode == "rotate" {
		return "rotate"
	}
	return "fixed"
}
func (c *codexTicketConfig) interval() time.Duration {
	if c.ProbeIntervalSeconds >= 6 && c.ProbeIntervalSeconds <= 3600 {
		return time.Duration(c.ProbeIntervalSeconds) * time.Second
	}
	return 6 * time.Second
}
func (c *codexTicketConfig) attempts() int {
	if c.unlimitedAttempts {
		return 0
	}
	if c.MaxAttempts >= 1 {
		return c.MaxAttempts
	}
	return 1
}

// 默认兼容旧配置；长度上限仅限制可注入 HTTP 头的大小，不按经验猜测票据意义。
func (c *codexTicketConfig) targetLength() int {
	if c.TargetLength >= 6 && c.TargetLength <= 8192 {
		return c.TargetLength
	}
	return 292
}

// 缺省模型保持旧配置兼容；实际模型支持仍由账号与上游判断，不根据名字猜测能力。
func (c *codexTicketConfig) models() []string {
	if len(c.Models) > 0 {
		return c.Models
	}
	return []string{"gpt-6-astra", "gpt-5.6-sol"}
}
func (c *codexTicketConfig) hasModel(model string) bool {
	for _, configured := range c.models() {
		if model == configured {
			return true
		}
	}
	return false
}
func ValidCodexTicketModelID(model string) bool {
	return model != "" && len(model) <= 256 && strings.IndexFunc(model, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) < 0
}
func (c *codexTicketConfig) retryInterval() time.Duration {
	if c.RetryIntervalSeconds >= 1 && c.RetryIntervalSeconds <= 30 {
		return time.Duration(c.RetryIntervalSeconds) * time.Second
	}
	return time.Second
}
func codexTicketSettingsView(c *codexTicketConfig) CodexTicketSettings {
	v := CodexTicketSettings{Enabled: c.Enabled, ProxyConfigured: len(c.proxies()) > 0, Proxies: []CodexTicketProxyView{}, SelectionMode: c.mode(), FixedProxyID: c.FixedProxyID, ProbeIntervalSeconds: int(c.interval() / time.Second), MaxAttempts: c.attempts(), RetryIntervalSeconds: int(c.retryInterval() / time.Second), Revision: c.Generation}
	v.TargetLength = c.targetLength()
	v.ProxyPolicy = ticketProxyPolicyView(c)
	v.WatchdogMode = ticketWatchdogMode(c.WatchdogMode)
	for _, account := range c.Accounts {
		if account.Mode != "off" && (account.ProxyCipher != "" || account.ProxyPolicy != nil && account.ProxyPolicy.configured()) {
			v.AccountProxyConfigured = true
			break
		}
	}
	v.Models = append([]string(nil), c.models()...)
	v.DegradedSignalLength = c.DegradedSignalLength
	for _, p := range c.proxies() {
		v.Proxies = append(v.Proxies, CodexTicketProxyView{ID: p.ID, Name: p.Name, Configured: p.Cipher != "" || p.ManagedID > 0, ManagedID: p.ManagedID})
	}
	if v.FixedProxyID == "" && len(v.Proxies) > 0 {
		v.FixedProxyID = v.Proxies[0].ID
	}
	// 旧响应字段仍保留，但只投影唯一的新号模板，不能再成为隐式运行默认值。
	template := ticketAccountSettingsView(ticketTemplateConfig(c), 0)
	v.Models, v.TargetLength, v.DegradedSignalLength = template.Rules.Models, template.Rules.TargetLength, template.Rules.DegradedSignalLength
	v.MaxAttempts, v.ProbeIntervalSeconds, v.RetryIntervalSeconds = template.Rules.MaxAttempts, template.Rules.ProbeIntervalSeconds, template.Rules.RetryIntervalSeconds
	v.WatchdogMode = template.WatchdogMode
	return v
}

// 新界面明确提交整张列表；旧单代理字段仍可用，但不能不知情地覆盖多代理配置。
func (s *CodexTicketService) updateProxySettings(c *codexTicketConfig, u CodexTicketSettingsUpdate) error {
	if u.Revision != nil && *u.Revision != c.Generation {
		return errors.New("票据配置已更新，请重新加载后再保存")
	}
	if u.Proxies != nil && (u.ClearProxy || (u.HarvestProxyURL != nil && strings.TrimSpace(*u.HarvestProxyURL) != "")) {
		return errors.New("不能同时提交新代理列表与旧单代理字段")
	}
	old := c.proxies()
	// API取号provider是运行时占位，不是固定代理行，不能妨碍关闭总闸。
	if c.ProxyPolicy != nil {
		old = c.ProxyPolicy.Proxies
	}
	list := append([]codexTicketProxy(nil), old...)
	if u.Proxies != nil {
		if len(*u.Proxies) > 20 {
			return errors.New("最多配置 20 条采集代理")
		}
		byID := map[string]codexTicketProxy{}
		for _, p := range old {
			byID[p.ID] = p
		}
		seen := map[string]bool{}
		list = nil
		for _, p := range *u.Proxies {
			name := strings.TrimSpace(p.Name)
			if name == "" || utf8.RuneCountInString(name) > 64 || strings.ContainsAny(name, "\r\n\x00") {
				return errors.New("代理名称不能为空且最多 64 字")
			}
			if p.ManagedID < 0 || p.ManagedID > 0 && strings.TrimSpace(p.URL) != "" {
				return errors.New("每条采集代理只能选择管理代理或填写地址")
			}
			entry := codexTicketProxy{ID: p.ID, Name: name, ManagedID: p.ManagedID}
			if p.ID != "" {
				previous, ok := byID[p.ID]
				if seen[p.ID] {
					return errors.New("代理 ID 重复")
				}
				if ok {
					if p.ManagedID == 0 {
						entry.Cipher = previous.Cipher
					}
				} else if _, err := uuid.Parse(p.ID); err != nil || strings.TrimSpace(p.URL) == "" && p.ManagedID == 0 {
					return errors.New("代理 ID 无效，请重新加载")
				}
			} else {
				entry.ID = uuid.NewString()
			}
			seen[entry.ID] = true
			if raw := strings.TrimSpace(p.URL); raw != "" {
				encrypted, err := s.encryptHarvestProxy(raw)
				if err != nil {
					return err
				}
				entry.Cipher = encrypted
			}
			if entry.Cipher == "" && entry.ManagedID == 0 {
				return errors.New("新增代理必须填写地址")
			}
			list = append(list, entry)
		}
	} else if u.ClearProxy || (u.HarvestProxyURL != nil && strings.TrimSpace(*u.HarvestProxyURL) != "") {
		if len(old) > 1 {
			return errors.New("当前为多代理配置，请使用新版代理列表保存")
		}
		if u.ClearProxy {
			list = nil
		}
		if u.HarvestProxyURL != nil && strings.TrimSpace(*u.HarvestProxyURL) != "" {
			encrypted, err := s.encryptHarvestProxy(strings.TrimSpace(*u.HarvestProxyURL))
			if err != nil {
				return err
			}
			list = []codexTicketProxy{{ID: "legacy", Name: "Proxy 1", Cipher: encrypted}}
		}
		c.FixedProxyID = ""
	}
	if u.SelectionMode != nil {
		if *u.SelectionMode != "fixed" && *u.SelectionMode != "rotate" {
			return errors.New("无效的代理选择模式")
		}
		c.SelectionMode = *u.SelectionMode
	}
	if u.FixedProxyID != nil {
		c.FixedProxyID = *u.FixedProxyID
	}
	if len(list) == 0 {
		c.FixedProxyID = ""
	} else {
		if c.FixedProxyID == "" {
			c.FixedProxyID = list[0].ID
		}
		found := false
		for _, p := range list {
			if p.ID == c.FixedProxyID {
				found = true
			}
		}
		if !found {
			return errors.New("固定代理不存在，请重新选择")
		}
	}
	c.Proxies = list
	c.ProxyCipher = ""
	// 回退旧版本时只使用当前指定的一条，不把列表序列化成旧代理地址。
	for _, p := range list {
		if p.ID == c.FixedProxyID {
			c.ProxyCipher = p.Cipher
		}
	}
	return nil
}
func (s *CodexTicketService) encryptHarvestProxy(raw string) (string, error) {
	if err := validateCodexHarvestProxy(expandTicketProxySession(raw)); err != nil {
		return "", err
	}
	if s.cipher == nil {
		return "", errors.New("代理加密服务不可用")
	}
	value, err := s.cipher.Encrypt(raw)
	if err != nil {
		return "", errors.New("保存代理凭据失败")
	}
	return value, nil
}

// 成功沿用该代理，失败才在下次尝试选择下一项；轮换依据每个账号/模型的记录。
func selectCodexTicketProxy(c *codexTicketConfig, previousID string, failed bool) (codexTicketProxy, bool) {
	list := c.proxies()
	if len(list) == 0 {
		return codexTicketProxy{}, false
	}
	if c.mode() == "fixed" {
		for _, p := range list {
			if p.ID == c.FixedProxyID || (c.FixedProxyID == "" && p.ID == list[0].ID) {
				return p, true
			}
		}
		return codexTicketProxy{}, false
	}
	for i, p := range list {
		if p.ID == previousID {
			if failed {
				return list[(i+1)%len(list)], true
			}
			return p, true
		}
	}
	return list[0], true
}
