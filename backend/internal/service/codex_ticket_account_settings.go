package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// 账号覆盖仅保存在私有票据设置中；代理密文不得进入账号extra、导出或调度摘要。
// @project-doc docs/interfaces/codex_ticket.md#account_overrides_watchdog
type codexTicketAccountConfig struct {
	VerifiedFlow bool                    `json:"verified_flow,omitempty"`
	Rules        *CodexTicketRules       `json:"rules,omitempty"`
	ProxyPolicy  *codexTicketProxyPolicy `json:"proxy_policy,omitempty"`
	Mode         string                  `json:"mode"`
	ProxyCipher  string                  `json:"proxy_cipher,omitempty"`
	WatchdogMode string                  `json:"watchdog_mode,omitempty"`
	Revision     string                  `json:"revision"`
}

type CodexTicketAccountSettings struct {
	VerifiedFlow          bool                       `json:"verified_flow"`
	GlobalEnabled         bool                       `json:"global_enabled"`
	Rules                 CodexTicketRules           `json:"rules"`
	ProxyPolicy           CodexTicketProxyPolicyView `json:"proxy_policy"`
	AccountID             int64                      `json:"account_id"`
	Mode                  string                     `json:"mode"`
	EffectiveEnabled      bool                       `json:"effective_enabled"`
	ProxyConfigured       bool                       `json:"proxy_configured"`
	ProxySource           string                     `json:"proxy_source"`
	WatchdogMode          string                     `json:"watchdog_mode"`
	EffectiveWatchdogMode string                     `json:"effective_watchdog_mode"`
	Revision              string                     `json:"revision"`
}

// 指针缺省意味着不修改；代理空串恢复继承，未提供意味着保留原密文。
type CodexTicketAccountPatch struct {
	VerifiedFlow    *bool                         `json:"verified_flow"`
	Rules           *CodexTicketRulesPatch        `json:"rules"`
	ProxyPolicy     *CodexTicketProxyPolicyUpdate `json:"proxy_policy"`
	Mode            *string                       `json:"mode"`
	HarvestProxyURL *string                       `json:"harvest_proxy_url"`
	WatchdogMode    *string                       `json:"watchdog_mode"`
}

type CodexTicketAccountsUpdate struct {
	AccountIDs []int64                 `json:"account_ids"`
	Patch      CodexTicketAccountPatch `json:"patch"`
	Revision   *string                 `json:"revision,omitempty"`
}

func validTicketWatchdogMode(mode string, inherit bool) bool {
	return mode == "off" || mode == "observe" || mode == "recover" || mode == "recover_length" || mode == "recover_model" || (inherit && mode == "inherit")
}
func ticketWatchdogMode(mode string) string {
	if validTicketWatchdogMode(mode, false) {
		return mode
	}
	return "observe"
}

// 全局开关是总闸；账号开启也不能越过总闸，未配置账号保持上一版的全局行为。
func ticketConfigForAccount(cfg *codexTicketConfig, id int64) *codexTicketConfig {
	if cfg == nil {
		return nil
	}
	if cfg.accountID != 0 {
		if cfg.accountID != id {
			copy := *cfg
			copy.Enabled = false
			return &copy
		}
		return cfg
	}
	copy := *cfg
	copy.accountID = id
	override := canonicalTicketAccount(cfg, cfg.Accounts[strconv.FormatInt(id, 10)])
	if override.Mode == "off" {
		copy.Enabled = false
	}
	if override.Revision != "" {
		copy.Generation += ":" + override.Revision
	}
	if override.Rules != nil {
		copy.applyRules(*override.Rules)
	}
	copy.VerifiedFlow = override.VerifiedFlow
	if override.ProxyPolicy != nil {
		copy.applyProxyPolicy(override.ProxyPolicy)
	} else if cfg.ProxyPolicy != nil {
		copy.applyProxyPolicy(cfg.ProxyPolicy)
	}
	copy.WatchdogMode = override.WatchdogMode
	// 双链路模式包含异常自动废票重采，关闭后恢复原先保存的守护选择。
	if copy.VerifiedFlow {
		copy.WatchdogMode = "recover"
	}
	return &copy
}

func (s *CodexTicketService) enabledAccountConfig(id int64) *codexTicketConfig {
	cfg := ticketConfigForAccount(s.enabledConfig(), id)
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	return cfg
}
func (s *CodexTicketService) ticketConfigCurrent(cfg *codexTicketConfig) bool {
	if cfg == nil {
		return false
	}
	current := s.enabledConfig()
	if cfg.accountID > 0 {
		current = s.enabledAccountConfig(cfg.accountID)
	}
	return current != nil && current.Generation == cfg.Generation
}

func ticketAccountSettingsView(cfg *codexTicketConfig, id int64) CodexTicketAccountSettings {
	a := canonicalTicketAccount(cfg, cfg.Accounts[strconv.FormatInt(id, 10)])
	mode, guard := a.Mode, a.WatchdogMode
	effective := ticketConfigForAccount(cfg, id)
	source := "gateway"
	if a.ProxyPolicy != nil {
		source = "account"
	}
	return CodexTicketAccountSettings{AccountID: id, Mode: mode, EffectiveEnabled: effective.Enabled,
		VerifiedFlow:  effective.VerifiedFlow,
		GlobalEnabled: cfg.Enabled,
		Rules:         ticketRulesFromConfig(effective), ProxyPolicy: ticketProxyPolicyView(effective),
		ProxyConfigured: source == "account", ProxySource: source, WatchdogMode: guard,
		EffectiveWatchdogMode: ticketWatchdogMode(effective.WatchdogMode), Revision: a.Revision}
}

// 数据库CAS避免多实例账号编辑/全局保存互相覆盖，不依赖Redis，缓存故障时仍能关闭总开关。
type CodexTicketSettingsStore interface {
	CompareAndSwapTicketSettings(context.Context, *string, string) (bool, error)
}

func (s *CodexTicketService) persistTicketConfig(ctx context.Context, cfg *codexTicketConfig, raw string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if repo, ok := s.settings.(CodexTicketSettingsStore); ok {
		saved, err := repo.CompareAndSwapTicketSettings(ctx, cfg.stored, raw)
		if err != nil {
			return errors.New("保存票据配置失败")
		}
		if !saved {
			return errors.New("票据配置已被其他操作更新，请重新加载后保存")
		}
		return nil
	}
	// 非数据库测试替身仍由本服务updateMu串行；生产仓储必须实现CAS。
	return s.settings.Set(ctx, codexTicketSettingsKey, raw)
}

func (s *CodexTicketService) AccountSettings(ctx context.Context, id int64) (CodexTicketAccountSettings, error) {
	if s == nil || s.gateway == nil || s.gateway.accountRepo == nil {
		return CodexTicketAccountSettings{}, errors.New("票据服务不可用")
	}
	a, err := s.gateway.accountRepo.GetByID(ctx, id)
	if err != nil || !codexTicketAccount(a) {
		return CodexTicketAccountSettings{}, errors.New("仅支持独立OpenAI OAuth账号")
	}
	cfg, err := s.readConfig(ctx)
	if err != nil {
		return CodexTicketAccountSettings{}, err
	}
	return ticketAccountSettingsView(cfg, id), nil
}

// 先校验整批账号与配置，再单次持久化，失败不返回部分成功；批量未勾选字段不影响原值。
func (s *CodexTicketService) UpdateAccountSettings(ctx context.Context, input CodexTicketAccountsUpdate) ([]CodexTicketAccountSettings, error) {
	if s == nil || s.gateway == nil || s.gateway.accountRepo == nil {
		return nil, errors.New("票据服务不可用")
	}
	if len(input.AccountIDs) < 1 || len(input.AccountIDs) > 500 {
		return nil, errors.New("每批选择1–500个账号")
	}
	p := input.Patch
	if p.Mode == nil && p.HarvestProxyURL == nil && p.WatchdogMode == nil && p.Rules == nil && p.ProxyPolicy == nil && p.VerifiedFlow == nil {
		return nil, errors.New("请勾选要修改的项目")
	}
	if p.Mode != nil && *p.Mode != "inherit" && *p.Mode != "on" && *p.Mode != "off" {
		return nil, errors.New("无效的账号票据模式")
	}
	if p.WatchdogMode != nil && !validTicketWatchdogMode(*p.WatchdogMode, true) {
		return nil, errors.New("无效的守护模式")
	}
	proxyCipher := ""
	if p.HarvestProxyURL != nil && strings.TrimSpace(*p.HarvestProxyURL) != "" {
		proxy := strings.TrimSpace(*p.HarvestProxyURL)
		if validateCodexHarvestProxy(expandTicketProxySession(proxy)) != nil || len(proxy) > 4096 {
			return nil, errors.New("无效的采集代理")
		}
		if s.cipher == nil {
			return nil, errors.New("代理加密服务不可用")
		}
		var err error
		proxyCipher, err = s.cipher.Encrypt(proxy)
		if err != nil {
			return nil, errors.New("代理加密失败")
		}
	}
	seen := map[int64]bool{}
	ids := []int64{}
	for _, id := range input.AccountIDs {
		if id <= 0 {
			return nil, errors.New("账号ID无效")
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	accounts, err := s.gateway.accountRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, errors.New("读取账号失败")
	}
	valid := map[int64]bool{}
	for _, a := range accounts {
		if codexTicketAccount(a) {
			valid[a.ID] = true
		}
	}
	for _, id := range ids {
		if !valid[id] {
			return nil, errors.New("所选账号包含不存在或不支持的账号；本批未修改")
		}
	}
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	cfg, err := s.readConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.Accounts == nil {
		cfg.Accounts = map[string]codexTicketAccountConfig{}
	}
	for _, id := range ids {
		key := strconv.FormatInt(id, 10)
		a := canonicalTicketAccount(cfg, cfg.Accounts[key])
		if input.Revision != nil && (len(ids) != 1 || a.Revision != *input.Revision) {
			return nil, errors.New("账号票据配置已变化，请重新加载")
		}
		before := a
		if p.VerifiedFlow != nil {
			a.VerifiedFlow = *p.VerifiedFlow
		}
		rules, e := applyTicketRulesPatch(ticketRulesFromConfig(ticketConfigForAccount(cfg, id)), p.Rules)
		if e != nil {
			return nil, e
		}
		a.Rules = &rules
		if p.ProxyPolicy != nil {
			if p.HarvestProxyURL != nil {
				return nil, errors.New("不能同时提交旧代理地址与新代理配置")
			}
			if p.ProxyPolicy.Mode == "inherit" {
				a.ProxyPolicy = nil
				a.ProxyCipher = ""
			} else {
				policy, e := s.updateTicketProxyPolicy(ticketConfigForAccount(cfg, id), p.ProxyPolicy)
				if e != nil {
					return nil, e
				}
				a.ProxyPolicy = policy
				a.ProxyCipher = ""
			}
		}
		if p.Mode != nil {
			a.Mode = *p.Mode
		}
		if p.WatchdogMode != nil {
			a.WatchdogMode = *p.WatchdogMode
		}
		if p.HarvestProxyURL != nil {
			a.ProxyCipher = proxyCipher
			a.ProxyPolicy = nil
		}
		a = canonicalTicketAccount(cfg, a)
		if !reflect.DeepEqual(a, before) {
			a.Revision = uuid.NewString()
		}
		cfg.Accounts[key] = a
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, errors.New("编码票据设置失败")
	}
	if err = s.persistTicketConfig(ctx, cfg, string(raw)); err != nil {
		return nil, err
	}
	s.publishConfig(cfg)
	result := make([]CodexTicketAccountSettings, 0, len(ids))
	for _, id := range ids {
		result = append(result, ticketAccountSettingsView(cfg, id))
	}
	return result, nil
}

// 只替换代理认证信息中的显式占位符，不推断服务商、不改变业务代理；SID并不保证出口不同。
func expandTicketProxySession(raw string) string {
	escaped := strings.NewReplacer("{sid}", "%7Bsid%7D", "{random}", "%7Brandom%7D").Replace(raw)
	u, err := url.Parse(escaped)
	if err != nil || u.User == nil {
		return raw
	}
	sid := strings.ReplaceAll(uuid.NewString(), "-", "")
	replace := strings.NewReplacer("{sid}", sid, "{random}", sid)
	username := replace.Replace(u.User.Username())
	if password, ok := u.User.Password(); ok {
		u.User = url.UserPassword(username, replace.Replace(password))
	} else {
		u.User = url.User(username)
	}
	return u.String()
}

// 手动一次尝试的采集与参考IP查询使用同一SID；参考查询仍是独立连接，不承诺真实出口相同。
func (s *CodexTicketService) ticketAttemptProxy(proxy codexTicketProxy) (codexTicketProxy, error) {
	raw, err := s.cipher.Decrypt(proxy.Cipher)
	if err != nil {
		return proxy, errors.New("采集代理不可读")
	}
	expanded := expandTicketProxySession(raw)
	if raw == expanded {
		return proxy, nil
	}
	proxy.Cipher, err = s.cipher.Encrypt(expanded)
	return proxy, err
}
