package composite

import "github.com/TokenFlux/TokenRouter/internal/provider"

// ApplyProviderAdminReadSettings 仅转换所属模块值，不读取设置或发布状态。
func (s *Snapshot) ApplyProviderAdminReadSettings(value *provider.AdminReadSettings) {
	s.ProviderQuotaNotifyEmails = value.ProviderQuotaNotifyEmails
	s.ProviderQuotaNotifyEnabled = value.ProviderQuotaNotifyEnabled
	s.ProviderSchedulingThresholds = value.ProviderSchedulingThresholds
}
