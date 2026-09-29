package billing

import (
	"strconv"
	"strings"

	settingvalues "github.com/TokenFlux/TokenRouter/internal/settings"
)

// AdminReadSettings 只包含本模块在综合管理页的展示投影。
type AdminReadSettings struct {
	BalanceIconSVG                  string
	BalanceLowNotifyEnabled         bool
	BalanceLowNotifyRechargeURL     string
	BalanceLowNotifyThreshold       float64
	BalanceUnitName                 string
	BalanceUnitSymbol               string
	DefaultBalance                  float64
	DefaultSubscriptions            []DefaultSubscriptionSetting
	ReasoningPointRMBUnitPrice      float64
	SubscriptionExpiryNotifyEnabled bool
	USDExchangeRate                 float64
}

// ReadAdminSettings 解释同一批已读持久值，不新增查询或改变缺省语义。
func ReadAdminSettings(settings map[string]string, defaultBalance func() float64) *AdminReadSettings {
	balanceUnitName := strings.TrimSpace(settings[SettingKeyBalanceUnitName])
	if balanceUnitName == "" {
		balanceUnitName = "USD"
	}
	balanceUnitSymbol := strings.TrimSpace(settings[SettingKeyBalanceUnitSymbol])
	if balanceUnitSymbol == "" {
		balanceUnitSymbol = "$"
	}
	result := &AdminReadSettings{}
	result.BalanceUnitName = balanceUnitName
	result.BalanceUnitSymbol = balanceUnitSymbol
	result.BalanceIconSVG = strings.TrimSpace(settings[SettingKeyBalanceIconSVG])
	if balance, err := strconv.ParseFloat(settings[SettingKeyDefaultBalance], 64); err == nil {
		result.DefaultBalance = balance
	} else {
		result.DefaultBalance = defaultBalance()
	}
	if price, err := strconv.ParseFloat(settings[SettingKeyReasoningPointRMBUnitPrice], 64); err == nil && price >= 0 {
		result.ReasoningPointRMBUnitPrice = price
	}
	if rate, err := strconv.ParseFloat(settings[SettingKeyUSDExchangeRate], 64); err == nil && rate >= 0 {
		result.USDExchangeRate = rate
	}
	result.DefaultSubscriptions = ParseDefaultSubscriptions(settings[SettingKeyDefaultSubscriptions])
	result.BalanceLowNotifyEnabled = settings[SettingKeyBalanceLowNotifyEnabled] == "true"
	if v, err := strconv.ParseFloat(settings[SettingKeyBalanceLowNotifyThreshold], 64); err == nil && v >= 0 {
		result.BalanceLowNotifyThreshold = v
	}
	result.BalanceLowNotifyRechargeURL = settings[SettingKeyBalanceLowNotifyRechargeURL]
	result.SubscriptionExpiryNotifyEnabled = !settingvalues.IsExplicitFalse(settings[SettingKeySubscriptionExpiryNotifyEnabled])
	return result
}
