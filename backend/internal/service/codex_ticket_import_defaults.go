package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/google/uuid"
)

// 默认模板仍保存在私有票据设置，不进入普通账号extra或凭据导出。
type CodexTicketAccountCreation struct {
	Expected             *string
	Configuration        string
	AccountConfiguration string
}

type CodexTicketAccountCreator interface {
	CreateWithCodexTicket(context.Context, *Account, []int64, CodexTicketAccountCreation) error
}

func (s *OpenAIGatewayService) CodexTicketConfiguration() *CodexTicketService {
	return s.codexTickets.Load()
}

// 用虚拟ID生成模板的有效视图，真实配置的账号映射不被改动。
func ticketTemplateConfig(cfg *codexTicketConfig) *codexTicketConfig {
	copy := *cfg
	copy.Accounts = map[string]codexTicketAccountConfig{}
	template := codexTicketAccountConfig{}
	if cfg.ImportDefaults != nil {
		template = *cfg.ImportDefaults
	}
	copy.Accounts["0"] = canonicalTicketAccount(cfg, template)
	return &copy
}

func (s *CodexTicketService) ImportDefaults(ctx context.Context) (CodexTicketAccountSettings, error) {
	cfg, err := s.readConfig(ctx)
	if err != nil {
		return CodexTicketAccountSettings{}, err
	}
	return ticketAccountSettingsView(ticketTemplateConfig(cfg), 0), nil
}

// 共用既有字段校验和代理加密，缺省字段保留模板；0不限等显式零值不会丢失。
func (s *CodexTicketService) applyImportPatch(cfg *codexTicketConfig, p *CodexTicketAccountPatch) (codexTicketAccountConfig, error) {
	view := ticketTemplateConfig(cfg)
	a := view.Accounts["0"]
	if p == nil {
		return a, nil
	}
	if p.Mode != nil {
		if *p.Mode != "inherit" && *p.Mode != "on" && *p.Mode != "off" {
			return a, errors.New("无效的账号票据模式")
		}
		a.Mode = *p.Mode
	}
	if p.WatchdogMode != nil {
		if !validTicketWatchdogMode(*p.WatchdogMode, true) {
			return a, errors.New("无效的守护模式")
		}
		a.WatchdogMode = *p.WatchdogMode
	}
	if p.HarvestProxyURL != nil {
		return a, errors.New("新增票据请使用采集代理配置")
	}
	if p.VerifiedFlow != nil {
		a.VerifiedFlow = *p.VerifiedFlow
	}
	rules, err := applyTicketRulesPatch(ticketRulesFromConfig(ticketConfigForAccount(view, 0)), p.Rules)
	if err != nil {
		return a, err
	}
	a.Rules = &rules
	if p.ProxyPolicy != nil {
		if p.ProxyPolicy.Mode == "inherit" {
			a.ProxyPolicy, a.ProxyCipher = nil, ""
		} else {
			policy, err := s.updateTicketProxyPolicy(ticketConfigForAccount(view, 0), p.ProxyPolicy)
			if err != nil {
				return a, err
			}
			a.ProxyPolicy, a.ProxyCipher = policy, ""
		}
	}
	a.Revision = uuid.NewString()
	return canonicalTicketAccount(cfg, a), nil
}

func (s *CodexTicketService) UpdateImportDefaults(ctx context.Context, patch CodexTicketAccountPatch, revision *string) (CodexTicketAccountSettings, error) {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	cfg, err := s.readConfig(ctx)
	if err != nil {
		return CodexTicketAccountSettings{}, err
	}
	before := ticketAccountSettingsView(ticketTemplateConfig(cfg), 0)
	if revision != nil && before.Revision != *revision {
		return before, errors.New("默认模板已变化，请重新加载")
	}
	a, err := s.applyImportPatch(cfg, &patch)
	if err != nil {
		return before, err
	}
	cfg.ImportDefaults = &a
	raw, err := json.Marshal(cfg)
	if err != nil {
		return before, err
	}
	if err := s.persistTicketConfig(ctx, cfg, string(raw)); err != nil {
		return before, err
	}
	s.publishConfig(cfg)
	return ticketAccountSettingsView(ticketTemplateConfig(cfg), 0), nil
}

// 只在新账号入口执行；重导入已有账号不覆盖其票据。账号/分组/模板副本必须原子提交。
func (s *CodexTicketService) CreateAccountWithDefaults(ctx context.Context, account *Account, groups []int64, patch *CodexTicketAccountPatch) (bool, error) {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	cfg, err := s.readConfig(ctx)
	if err != nil {
		return true, err
	}
	// 所有新OAuth账号都固化模板；旧调用方缺省字段也不能创建会长期继承旧网关规则的新号。
	a, err := s.applyImportPatch(cfg, patch)
	if err != nil {
		return true, err
	}
	a.Revision = uuid.NewString()
	repo, ok := s.gateway.accountRepo.(CodexTicketAccountCreator)
	if !ok {
		return true, errors.New("账号票据原子创建不可用")
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return true, err
	}
	accountRaw, err := json.Marshal(a)
	if err != nil {
		return true, err
	}
	if err := repo.CreateWithCodexTicket(ctx, account, groups, CodexTicketAccountCreation{Expected: cfg.stored, Configuration: string(raw), AccountConfiguration: string(accountRaw)}); err != nil {
		return true, err
	}
	if cfg.Accounts == nil {
		cfg.Accounts = map[string]codexTicketAccountConfig{}
	}
	cfg.Accounts[strconv.FormatInt(account.ID, 10)] = a
	s.publishConfig(cfg)
	return true, nil
}
