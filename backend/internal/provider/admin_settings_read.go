package provider

import (
	"log/slog"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/identity/contact"
)

// AdminReadSettings 只包含本模块在综合管理页的展示投影。
type AdminReadSettings struct {
	ProviderQuotaNotifyEmails    []contact.Entry
	ProviderQuotaNotifyEnabled   bool
	ProviderSchedulingThresholds map[string]int
}

// ReadAdminSettings 解释同一批已读持久值，不新增查询或改变缺省语义。
func ReadAdminSettings(settings map[string]string) *AdminReadSettings {
	result := &AdminReadSettings{}

	result.ProviderQuotaNotifyEnabled = settings[SettingKeyProviderQuotaNotifyEnabled] == "true"
	if raw := strings.TrimSpace(settings[SettingKeyProviderQuotaNotifyEmails]); raw != "" {
		result.ProviderQuotaNotifyEmails = contact.ParseNotifyEmails(raw)
	}
	if result.ProviderQuotaNotifyEmails == nil {
		result.ProviderQuotaNotifyEmails = []contact.Entry{}
	}
	result.ProviderSchedulingThresholds = DefaultProviderSchedulingThresholds()
	if raw := strings.TrimSpace(settings[SettingKeyProviderSchedulingThresholds]); raw != "" {
		if thresholds, err := ParseProviderSchedulingThresholdsSetting(raw); err != nil {
			slog.Warn("[Setting] parseSettings: unmarshal provider_scheduling_thresholds failed", "error", err)
		} else {
			result.ProviderSchedulingThresholds = thresholds
		}
	}
	return result
}
