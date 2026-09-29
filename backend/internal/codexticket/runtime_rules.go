package codexticket

import (
	"time"
)

// 兼容原管理API与仓储调用名，规则的唯一实现迁移到独立票据模块。
type CodexTicketRules = Rules
type CodexTicketRulesPatch = Patch

func ticketRulesFromConfig(c *codexTicketConfig) CodexTicketRules {
	return CodexTicketRules{AttemptTimeoutSeconds: int(c.attemptTimeout() / time.Second), FailureThreshold: c.FailureThreshold, CooldownSeconds: c.cooldownSeconds(), Models: append([]string(nil), c.models()...), TargetLength: c.targetLength(), DegradedSignalLength: c.DegradedSignalLength,
		MaxAttempts: c.attempts(), Concurrency: c.collectionConcurrency(), CacheMinutes: int(c.ticketTTL() / time.Minute),
		RefreshBeforeMinutes: int(c.refreshBefore() / time.Minute), RetryIntervalSeconds: int(c.retryInterval() / time.Second), ProbeIntervalSeconds: int(c.interval() / time.Second)}
}
func (c *codexTicketConfig) applyRules(r CodexTicketRules) {
	c.AttemptTimeoutSeconds = r.AttemptTimeoutSeconds
	c.FailureThreshold = r.FailureThreshold
	c.CooldownSeconds = r.CooldownSeconds
	c.Models = append([]string(nil), r.Models...)
	c.TargetLength = r.TargetLength
	c.DegradedSignalLength = r.DegradedSignalLength
	c.MaxAttempts = r.MaxAttempts
	c.unlimitedAttempts = r.MaxAttempts == 0
	c.CollectionConcurrency = r.Concurrency
	c.CacheMinutes = r.CacheMinutes
	c.RefreshBeforeMinutes = &r.RefreshBeforeMinutes
	c.RetryIntervalSeconds = r.RetryIntervalSeconds
	c.ProbeIntervalSeconds = r.ProbeIntervalSeconds
}
func (c *codexTicketConfig) collectionConcurrency() int {
	if c.CollectionConcurrency >= 1 && c.CollectionConcurrency <= 4 {
		return c.CollectionConcurrency
	}
	return 4
}

// 旧配置缺项仍为25秒；新写入只允许5–300秒，不提供无限占槽的超时设置。
func (c *codexTicketConfig) attemptTimeout() time.Duration {
	if c != nil && c.AttemptTimeoutSeconds >= 5 && c.AttemptTimeoutSeconds <= 300 {
		return time.Duration(c.AttemptTimeoutSeconds) * time.Second
	}
	return 25 * time.Second
}

// 租约覆盖采集总预算和有界资格复核/存票，不能在慢请求尚未结束时释放并发保护。
func (c *codexTicketConfig) collectionLeaseTTL() time.Duration {
	return c.attemptTimeout() + 20*time.Second
}

// 单号保留原30秒基础轮次；全局另给30秒派发窗口，已派发请求仍有完整预算和收尾余量。
func (c *codexTicketConfig) roundTimeout() time.Duration {
	budget := 25 * time.Second
	if c.accountID != 0 {
		if c.attemptTimeout() > budget {
			budget = c.attemptTimeout()
		}
		return budget + 5*time.Second
	}
	if d := ticketConfigForAccount(c, 0).attemptTimeout(); d > budget {
		budget = d
	}
	for id := range c.Accounts {
		account := c.Accounts[id]
		if account.Mode != "off" && account.Rules != nil {
			if d := (&codexTicketConfig{AttemptTimeoutSeconds: account.Rules.AttemptTimeoutSeconds}).attemptTimeout(); d > budget {
				budget = d
			}
		}
	}
	return budget + 35*time.Second
}
func (c *codexTicketConfig) ticketTTL() time.Duration {
	if c.CacheMinutes >= 1 && c.CacheMinutes <= 1440 {
		return time.Duration(c.CacheMinutes) * time.Minute
	}
	return time.Hour
}
func (c *codexTicketConfig) refreshBefore() time.Duration {
	if c.RefreshBeforeMinutes != nil {
		return time.Duration(*c.RefreshBeforeMinutes) * time.Minute
	}
	return 10 * time.Minute
}
func (c *codexTicketConfig) cooldownSeconds() int {
	if c.CooldownSeconds >= 1 && c.CooldownSeconds <= 86400 {
		return c.CooldownSeconds
	}
	return 300
}

// 全局只调度最早需要检查的账号，各账号仍由自己的probe租约限频；不能被旧网关长间隔拖慢。
func (c *codexTicketConfig) scanInterval() time.Duration {
	delay := c.interval()
	for _, a := range c.Accounts {
		if a.Mode != "off" && a.Rules != nil {
			d := time.Duration(a.Rules.ProbeIntervalSeconds) * time.Second
			if d >= 6*time.Second && d < delay {
				delay = d
			}
		}
	}
	return delay
}

func applyTicketRulesPatch(r CodexTicketRules, p *CodexTicketRulesPatch) (CodexTicketRules, error) {
	return ApplyPatch(r, p)
}
