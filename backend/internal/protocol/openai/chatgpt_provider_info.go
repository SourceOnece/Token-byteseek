// ChatGPT 元数据保留来源账户，以便提供商用例区分个人套餐和 workspace。
package openai

// ChatGPTAccountInfo 从 chatgpt.com/backend-api/accounts/check 获取的账户信息
type ChatGPTAccountInfo struct {
	PlanType string
	Email    string
	// ProviderID 优先取 account.account_id，缺失时使用 accounts 的 map key。
	// accounts/check 可能同时返回个人账户与 workspace，调用方据此区分数据来源。
	ProviderID            string
	SubscriptionExpiresAt string // entitlement.expires_at (RFC3339)
}
