package service

// 历史规则只用于无独立配置旧号的不可变兼容快照，不是可编辑的网关默认值。
// 新号总是复制ImportDefaults，修改模板不回写已有账号。
// @project-doc docs/interfaces/codex_ticket.md#account_overrides_watchdog
func legacyTicketAccountDefaults(cfg *codexTicketConfig) codexTicketAccountConfig {
	if cfg.LegacyAccountDefaults != nil && cfg.LegacyAccountDefaults.Rules != nil {
		return *cfg.LegacyAccountDefaults
	}
	rules := ticketRulesFromConfig(cfg)
	return codexTicketAccountConfig{Mode: "on", WatchdogMode: ticketWatchdogMode(cfg.WatchdogMode), Rules: &rules}
}

func canonicalTicketAccount(cfg *codexTicketConfig, account codexTicketAccountConfig) codexTicketAccountConfig {
	baseline := legacyTicketAccountDefaults(cfg)
	if account.Mode != "off" {
		account.Mode = "on"
	}
	if account.WatchdogMode == "" || account.WatchdogMode == "inherit" {
		// 必须保留双链路强制recover之前的原选择，不能用effective_watchdog_mode迁移。
		account.WatchdogMode = ticketWatchdogMode(baseline.WatchdogMode)
	}
	if account.Rules == nil {
		rules := *baseline.Rules
		rules.Models = append([]string(nil), baseline.Rules.Models...)
		account.Rules = &rules
	}
	// 新增字段缺项补旧固定25秒，不修改版本/票据键或共享的规则对象。
	if account.Rules.AttemptTimeoutSeconds == 0 {
		rules := *account.Rules
		rules.AttemptTimeoutSeconds = 25
		account.Rules = &rules
	}
	if account.ProxyCipher != "" {
		if account.ProxyPolicy == nil {
			account.ProxyPolicy = &codexTicketProxyPolicy{Mode: "fixed", DynamicSource: "template", ProxyProtocol: "http", FixedProxyID: "account", Proxies: []codexTicketProxy{{ID: "account", Name: "Account", Cipher: account.ProxyCipher}}}
		}
		account.ProxyCipher = ""
	}
	return account
}

// 归一化不写accounts、不修改代次；只在既有保存事务中持久化，旧票据键保持不变。
func normalizeTicketConfiguration(cfg *codexTicketConfig) {
	if cfg.LegacyAccountDefaults == nil {
		baseline := legacyTicketAccountDefaults(cfg)
		cfg.LegacyAccountDefaults = &baseline
	}
	// 旧单代理/列表只在读取边界转成统一策略；运行和写入都以proxy_policy为准。
	if cfg.ProxyPolicy == nil && len(cfg.proxies()) > 0 {
		cfg.ProxyPolicy = &codexTicketProxyPolicy{Mode: cfg.mode(), DynamicSource: "template", ProxyProtocol: "http", Proxies: append([]codexTicketProxy(nil), cfg.proxies()...), FixedProxyID: cfg.FixedProxyID}
	}
	if cfg.ProxyPolicy != nil {
		cfg.applyProxyPolicy(cfg.ProxyPolicy)
	}
	for id, account := range cfg.Accounts {
		cfg.Accounts[id] = canonicalTicketAccount(cfg, account)
	}
	template := codexTicketAccountConfig{}
	if cfg.ImportDefaults != nil {
		template = *cfg.ImportDefaults
	}
	template = canonicalTicketAccount(cfg, template)
	cfg.ImportDefaults = &template
}

// 旧网关参数只作为导入模板的写入适配，不再形成第二套运行规则。
func ticketLegacyTemplatePatch(input CodexTicketSettingsUpdate) *CodexTicketAccountPatch {
	if input.WatchdogMode == nil && input.Models == nil && input.TargetLength == nil && input.DegradedSignalLength == nil && input.MaxAttempts == nil && input.RetryIntervalSeconds == nil && input.ProbeIntervalSeconds == nil {
		return nil
	}
	return &CodexTicketAccountPatch{WatchdogMode: input.WatchdogMode, Rules: &CodexTicketRulesPatch{Models: input.Models, TargetLength: input.TargetLength, DegradedSignalLength: input.DegradedSignalLength, MaxAttempts: input.MaxAttempts, RetryIntervalSeconds: input.RetryIntervalSeconds, ProbeIntervalSeconds: input.ProbeIntervalSeconds}}
}

func ticketLegacyProxyPatch(input CodexTicketSettingsUpdate) bool {
	return input.HarvestProxyURL != nil || input.ClearProxy || input.Proxies != nil || input.SelectionMode != nil || input.FixedProxyID != nil
}
