package service

import (
	"errors"
	"strings"
	"time"
)

// 规则在账号中保存完整快照；缺少规则的历史账号沿旧值读取，迁移不擅自改变用户配置。
type CodexTicketRules struct {
	FailureThreshold     int      `json:"failure_threshold"`
	CooldownSeconds      int      `json:"cooldown_seconds"`
	Models               []string `json:"models"`
	TargetLength         int      `json:"target_length"`
	DegradedSignalLength int      `json:"degraded_signal_length"`
	MaxAttempts          int      `json:"max_attempts"`
	Concurrency          int      `json:"concurrency"`
	CacheMinutes         int      `json:"cache_minutes"`
	RefreshBeforeMinutes int      `json:"refresh_before_minutes"`
	RetryIntervalSeconds int      `json:"retry_interval_seconds"`
	ProbeIntervalSeconds int      `json:"probe_interval_seconds"`
}

// 每个字段独立可选，批量没有勾选的项目不能回填默认值。
type CodexTicketRulesPatch struct {
	FailureThreshold     *int      `json:"failure_threshold"`
	CooldownSeconds      *int      `json:"cooldown_seconds"`
	Models               *[]string `json:"models"`
	TargetLength         *int      `json:"target_length"`
	DegradedSignalLength *int      `json:"degraded_signal_length"`
	MaxAttempts          *int      `json:"max_attempts"`
	Concurrency          *int      `json:"concurrency"`
	CacheMinutes         *int      `json:"cache_minutes"`
	RefreshBeforeMinutes *int      `json:"refresh_before_minutes"`
	RetryIntervalSeconds *int      `json:"retry_interval_seconds"`
	ProbeIntervalSeconds *int      `json:"probe_interval_seconds"`
}

func ticketRulesFromConfig(c *codexTicketConfig) CodexTicketRules {
	return CodexTicketRules{FailureThreshold: c.FailureThreshold, CooldownSeconds: c.cooldownSeconds(), Models: append([]string(nil), c.models()...), TargetLength: c.targetLength(), DegradedSignalLength: c.DegradedSignalLength,
		MaxAttempts: c.attempts(), Concurrency: c.collectionConcurrency(), CacheMinutes: int(c.ticketTTL() / time.Minute),
		RefreshBeforeMinutes: int(c.refreshBefore() / time.Minute), RetryIntervalSeconds: int(c.retryInterval() / time.Second), ProbeIntervalSeconds: int(c.interval() / time.Second)}
}
func (c *codexTicketConfig) applyRules(r CodexTicketRules) {
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
	if p == nil {
		return r, nil
	}
	if p.Models != nil {
		models := []string{}
		seen := map[string]bool{}
		for _, v := range *p.Models {
			v = strings.TrimSpace(v)
			if !ValidCodexTicketModelID(v) {
				return r, errors.New("采集模型ID无效")
			}
			if !seen[v] {
				models = append(models, v)
				seen[v] = true
			}
		}
		if len(models) < 1 || len(models) > 100 {
			return r, errors.New("采集模型需要1–100个")
		}
		r.Models = models
	}
	for _, v := range []struct {
		input  *int
		target *int
		lo, hi int
	}{
		{p.FailureThreshold, &r.FailureThreshold, 0, 100000}, {p.CooldownSeconds, &r.CooldownSeconds, 1, 86400},
		{p.TargetLength, &r.TargetLength, 6, 8192}, {p.DegradedSignalLength, &r.DegradedSignalLength, 0, 8192},
		{p.MaxAttempts, &r.MaxAttempts, 0, 9007199254740991}, {p.Concurrency, &r.Concurrency, 1, 4},
		{p.CacheMinutes, &r.CacheMinutes, 1, 1440}, {p.RefreshBeforeMinutes, &r.RefreshBeforeMinutes, 0, 1439},
		{p.RetryIntervalSeconds, &r.RetryIntervalSeconds, 1, 30}, {p.ProbeIntervalSeconds, &r.ProbeIntervalSeconds, 6, 3600},
	} {
		if v.input != nil {
			if *v.input < v.lo || *v.input > v.hi {
				return r, errors.New("采集参数超出允许范围")
			}
			*v.target = *v.input
		}
	}
	if r.DegradedSignalLength > 0 && r.DegradedSignalLength < 6 {
		return r, errors.New("长度信号为0或6–8192")
	}
	if r.DegradedSignalLength > 0 && r.DegradedSignalLength == r.TargetLength {
		return r, errors.New("降智长度不能与合格长度相同")
	}
	if r.RefreshBeforeMinutes >= r.CacheMinutes {
		return r, errors.New("提前续采必须小于缓存时间")
	}
	return r, nil
}
