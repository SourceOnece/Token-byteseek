package service

import "github.com/TokenFlux/TokenRouter/internal/provider"

// ProviderRecord 是跨原生模块的显式核心投影，不包含旧分组对象和热路径缓存。
// 不做JSON中转，避免损失数值/时间精度；字段仍由当前调用拥有，不得在投影上并发修改共享配置。
func (a *Account) ProviderRecord() *provider.Record {
	if a == nil {
		return nil
	}
	return &provider.Record{
		ID: a.ID, Name: a.Name, Notes: a.Notes, Platform: a.Platform, Type: a.Type,
		Credentials: a.Credentials, Extra: a.Extra, Proxy: a.Proxy, ProxyID: a.ProxyID,
		ProxyFallbackOriginID: a.ProxyFallbackOriginID, ProxyFallbackOriginName: a.ProxyFallbackOriginName,
		Concurrency: a.Concurrency, Priority: a.Priority, RateMultiplier: a.RateMultiplier, LoadFactor: a.LoadFactor,
		Status: a.Status, ErrorMessage: a.ErrorMessage, LastUsedAt: a.LastUsedAt, ExpiresAt: a.ExpiresAt,
		AutoPauseOnExpired: a.AutoPauseOnExpired, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
		Schedulable: a.Schedulable, RateLimitedAt: a.RateLimitedAt, RateLimitResetAt: a.RateLimitResetAt,
		OverloadUntil: a.OverloadUntil, TempUnschedulableUntil: a.TempUnschedulableUntil,
		TempUnschedulableReason: a.TempUnschedulableReason, QuotaAutoPaused: a.QuotaAutoPaused,
		SessionWindowStart: a.SessionWindowStart, SessionWindowEnd: a.SessionWindowEnd, SessionWindowStatus: a.SessionWindowStatus,
		ParentProviderID: a.ParentAccountID, QuotaDimension: a.QuotaDimension, GroupIDs: a.GroupIDs,
	}
}

// AccountFromProviderRecord 将已读取的原生记录交回旧业务链；分组和缓存继续由原所有者补全。
// 不改变旧Account JSON外形，避免旧Redis缓存和票据摘要在迁移中丢失凭据。
func AccountFromProviderRecord(r *provider.Record) *Account {
	if r == nil {
		return nil
	}
	return &Account{
		ID: r.ID, Name: r.Name, Notes: r.Notes, Platform: r.Platform, Type: r.Type,
		Credentials: r.Credentials, Extra: r.Extra, Proxy: r.Proxy, ProxyID: r.ProxyID,
		ProxyFallbackOriginID: r.ProxyFallbackOriginID, ProxyFallbackOriginName: r.ProxyFallbackOriginName,
		Concurrency: r.Concurrency, Priority: r.Priority, RateMultiplier: r.RateMultiplier, LoadFactor: r.LoadFactor,
		Status: r.Status, ErrorMessage: r.ErrorMessage, LastUsedAt: r.LastUsedAt, ExpiresAt: r.ExpiresAt,
		AutoPauseOnExpired: r.AutoPauseOnExpired, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		Schedulable: r.Schedulable, RateLimitedAt: r.RateLimitedAt, RateLimitResetAt: r.RateLimitResetAt,
		OverloadUntil: r.OverloadUntil, TempUnschedulableUntil: r.TempUnschedulableUntil,
		TempUnschedulableReason: r.TempUnschedulableReason, QuotaAutoPaused: r.QuotaAutoPaused,
		SessionWindowStart: r.SessionWindowStart, SessionWindowEnd: r.SessionWindowEnd, SessionWindowStatus: r.SessionWindowStatus,
		ParentAccountID: r.ParentProviderID, QuotaDimension: r.QuotaDimension, GroupIDs: r.GroupIDs,
	}
}
