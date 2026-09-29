// 供应商额度查询结果用于提供商管理，不构成用户资金事实。
package provider

// QuotaResult 额度获取结果
type QuotaResult struct {
	UsageInfo *UsageInfo     // 转换后的使用信息
	Raw       map[string]any // 原始响应，可存入 provider.Extra
}
