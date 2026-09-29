package site

import (
	"strings"
	"time"
)

const SettingKeySubscriptionEnabled = "subscription_enabled"
const SettingBalancePayDisabled = "BALANCE_PAYMENT_DISABLED"

// 旧设置缺省保持订阅可用，已明确关闭的历史值仍被识别。
func isFalseSettingValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "false", "0", "off", "disabled":
		return true
	default:
		return false
	}
}

// 存储前拒绝无法用 JSON 返回的日期，nil 仍表示未设置。
func isJSONTimeInRange(value *time.Time) bool {
	return value == nil || value.Year() >= 0 && value.Year() <= 9999
}
