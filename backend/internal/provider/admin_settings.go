package provider

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/TokenFlux/TokenRouter/internal/identity/contact"
)

// AdminSettings 明确区分提供商健康阈值和通知目的地。
type AdminSettings struct {
	ProviderQuotaNotifyEnabled   bool            `json:"provider_quota_notify_enabled"`
	ProviderQuotaNotifyEmails    []contact.Entry `json:"provider_quota_notify_emails"`
	ProviderSchedulingThresholds map[string]int  `json:"provider_scheduling_thresholds"`
}

const (
	SettingKeyProviderQuotaNotifyEnabled = "provider_quota_notify_enabled"
	SettingKeyProviderQuotaNotifyEmails  = "provider_quota_notify_emails"
)

// PrepareAdminSettings 不执行处置或发送通知，保留旧邮箱序列化和 nil 阈值语义。
func PrepareAdminSettings(value AdminSettings) (map[string]string, error) {
	values := map[string]string{SettingKeyProviderQuotaNotifyEnabled: strconv.FormatBool(value.ProviderQuotaNotifyEnabled), SettingKeyProviderQuotaNotifyEmails: contact.MarshalNotifyEmails(value.ProviderQuotaNotifyEmails)}
	if value.ProviderSchedulingThresholds != nil {
		normalized, err := ValidateAndNormalizeProviderSchedulingThresholds(value.ProviderSchedulingThresholds)
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(normalized)
		if err != nil {
			return nil, fmt.Errorf("marshal provider scheduling thresholds: %w", err)
		}
		values[SettingKeyProviderSchedulingThresholds] = string(raw)
	}
	return values, nil
}
