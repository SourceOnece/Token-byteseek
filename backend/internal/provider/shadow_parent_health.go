package provider

// ParentHealthyForShadow 判断 Spark 影子共用的母提供商凭据是否可用于调度。
// 非影子直接返回 true；lookup 从调度快照或存储取得母提供商。
// 母提供商必须是 OpenAI OAuth、状态为 active、令牌未过期，且不处于 TempUnschedulableUntil 冷却期。
// 该冷却可能来自认证失败、刷新耗尽或传输故障，会影响共用凭据的影子。
// 母提供商的全局 RateLimitResetAt、OverloadUntil 和手动 Schedulable 开关不参与此判断，
// Spark 用量窗口独立维护。母提供商缺失、类型不符或凭据不可用时，影子不能进入候选池。
func ParentHealthyForShadow(provider *Record, lookup func(int64) *Record) bool {
	if provider == nil || !provider.IsShadow() {
		return true
	}
	parent := lookup(*provider.ParentProviderID)
	if parent == nil {
		return false
	}
	return parent.IsOpenAIOAuth() && parent.IsCredentialUsableForShadow()
}
