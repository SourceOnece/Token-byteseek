package provider

import (
	"time"
)

// GroupProviderCapacityRow 是容量汇总所需的轻量提供商投影。
type GroupProviderCapacityRow struct {
	GroupID             int64
	ProviderID          int64
	Platform            string
	Concurrency         int
	Extra               map[string]any
	SessionWindowStart  *time.Time
	SessionWindowEnd    *time.Time
	SessionWindowStatus string
}

// CapacitySnapshot 不包含凭据或原始 Extra，供路由聚合独立消费。
type CapacitySnapshot struct {
	ID                        int64
	Concurrency               int
	MaxSessions               int
	SessionIdleTimeoutMinutes int
	BaseRPM                   int
	QuotaAutoPaused           bool
}

func ProjectCapacity(id int64, config RuntimeConfig, quotaAutoPaused bool) CapacitySnapshot {
	return CapacitySnapshot{ID: id, Concurrency: config.Concurrency, MaxSessions: config.GetMaxSessions(), SessionIdleTimeoutMinutes: config.GetSessionIdleTimeoutMinutes(), BaseRPM: config.GetBaseRPM(), QuotaAutoPaused: quotaAutoPaused}
}

// ProjectObservedCapacity 组合提供商运行参数与纯阈值结果；调用方保留原逐行取时点。
func ProjectObservedCapacity(row GroupProviderCapacityRow, settings QuotaAutoPauseSettings, now time.Time) CapacitySnapshot {
	paused, _ := EvaluateQuotaAutoPause(row.Platform, row.Extra, settings, now)
	return ProjectCapacity(row.ProviderID, RuntimeConfig{Extra: row.Extra, Concurrency: row.Concurrency, SessionWindowStart: row.SessionWindowStart, SessionWindowEnd: row.SessionWindowEnd}, paused)
}
