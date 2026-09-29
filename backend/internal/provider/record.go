// Package provider 拥有上游账号的核心配置和运行记录。
package provider

import (
	"fmt"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/domain"
	"github.com/TokenFlux/TokenRouter/internal/egress"
)

// 迁移期与既有平台域共用状态值，避免两套字符串枚举漂移。
const StatusActive = domain.StatusActive

// Record 是内部核心记录，不是公开DTO或缓存编码。
// 凭据、扩展与代理不能经通用JSON/格式化输出；边界须使用专用脱敏DTO或私有缓存结构。
type Record struct {
	ID                      int64
	Name                    string
	Notes                   *string
	Platform                string
	Type                    string
	Credentials             map[string]any `json:"-"`
	Extra                   map[string]any `json:"-"`
	Proxy                   *egress.Proxy  `json:"-"`
	ProxyID                 *int64
	ProxyFallbackOriginID   *int64
	ProxyFallbackOriginName *string
	Concurrency             int
	Priority                int
	RateMultiplier          *float64
	LoadFactor              *int
	Status                  string
	ErrorMessage            string
	LastUsedAt              *time.Time
	ExpiresAt               *time.Time
	AutoPauseOnExpired      bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
	Schedulable             bool
	RateLimitedAt           *time.Time
	RateLimitResetAt        *time.Time
	OverloadUntil           *time.Time
	TempUnschedulableUntil  *time.Time
	TempUnschedulableReason string
	QuotaAutoPaused         bool `json:"-"`
	SessionWindowStart      *time.Time
	SessionWindowEnd        *time.Time
	SessionWindowStatus     string
	ParentProviderID        *int64
	QuotaDimension          string
	GroupIDs                []int64
}

// 只允许输出标识；用户名/备注也可能由管理员填入敏感材料，默认日志不展开。
func (r *Record) String() string {
	if r == nil {
		return "provider <nil>"
	}
	return fmt.Sprintf("provider (id=%d)", r.ID)
}
func (r *Record) GoString() string { return r.String() }

// BillingRateMultiplier 保留旧缓存缺省为1和显式0免费的账号成本语义。
func (r *Record) BillingRateMultiplier() float64 {
	if r == nil || r.RateMultiplier == nil || *r.RateMultiplier < 0 {
		return 1
	}
	return *r.RateMultiplier
}

// EffectiveLoadFactor 保留原并发负载分母，不将未知负载写成0。
func (r *Record) EffectiveLoadFactor() int {
	if r == nil {
		return 1
	}
	if r.LoadFactor != nil && *r.LoadFactor > 0 {
		return *r.LoadFactor
	}
	if r.Concurrency > 0 {
		return r.Concurrency
	}
	return 1
}

// SchedulingWindowOpen 只表达共同状态/时间门禁；额度、模型、票据和分组仍由调用链追加。
func (r *Record) SchedulingWindowOpen(now time.Time) bool {
	if r == nil || r.Status != StatusActive || !r.Schedulable {
		return false
	}
	if r.AutoPauseOnExpired && r.ExpiresAt != nil && !now.Before(*r.ExpiresAt) {
		return false
	}
	for _, until := range []*time.Time{r.OverloadUntil, r.RateLimitResetAt, r.TempUnschedulableUntil} {
		if until != nil && now.Before(*until) {
			return false
		}
	}
	return true
}
