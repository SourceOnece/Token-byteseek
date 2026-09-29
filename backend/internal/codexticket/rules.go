// Package codexticket 保存 ByteSeek 票据领域规则，不依赖网关、框架或数据库。
// @project-doc docs/interfaces/codex_ticket.md#ticket_runtime
package codexticket

import (
	"errors"
	"strings"
	"unicode"
)

// 规则在账号中保存完整快照；缺少规则的历史账号沿旧值读取，迁移不擅自改变用户配置。
type Rules struct {
	AttemptTimeoutSeconds int      `json:"attempt_timeout_seconds"`
	FailureThreshold      int      `json:"failure_threshold"`
	CooldownSeconds       int      `json:"cooldown_seconds"`
	Models                []string `json:"models"`
	TargetLength          int      `json:"target_length"`
	DegradedSignalLength  int      `json:"degraded_signal_length"`
	MaxAttempts           int      `json:"max_attempts"`
	Concurrency           int      `json:"concurrency"`
	CacheMinutes          int      `json:"cache_minutes"`
	RefreshBeforeMinutes  int      `json:"refresh_before_minutes"`
	RetryIntervalSeconds  int      `json:"retry_interval_seconds"`
	ProbeIntervalSeconds  int      `json:"probe_interval_seconds"`
}

// 每个字段独立可选，批量没有勾选的项目不能回填默认值。
type Patch struct {
	AttemptTimeoutSeconds *int      `json:"attempt_timeout_seconds"`
	FailureThreshold      *int      `json:"failure_threshold"`
	CooldownSeconds       *int      `json:"cooldown_seconds"`
	Models                *[]string `json:"models"`
	TargetLength          *int      `json:"target_length"`
	DegradedSignalLength  *int      `json:"degraded_signal_length"`
	MaxAttempts           *int      `json:"max_attempts"`
	Concurrency           *int      `json:"concurrency"`
	CacheMinutes          *int      `json:"cache_minutes"`
	RefreshBeforeMinutes  *int      `json:"refresh_before_minutes"`
	RetryIntervalSeconds  *int      `json:"retry_interval_seconds"`
	ProbeIntervalSeconds  *int      `json:"probe_interval_seconds"`
}

// ApplyPatch 仅修改显式提供的字段，保持未勾选值及零值的既有语义。
func ApplyPatch(r Rules, p *Patch) (Rules, error) {
	if p == nil {
		return r, nil
	}
	if p.Models != nil {
		models := []string{}
		seen := map[string]bool{}
		for _, v := range *p.Models {
			v = strings.TrimSpace(v)
			if !ValidModelID(v) {
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
		{p.AttemptTimeoutSeconds, &r.AttemptTimeoutSeconds, 5, 300},
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

// ValidModelID 校验可写入请求和诊断的精确模型名称，不推断模型权限。
func ValidModelID(model string) bool {
	return model != "" && len(model) <= 256 && strings.IndexFunc(model, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) < 0
}
