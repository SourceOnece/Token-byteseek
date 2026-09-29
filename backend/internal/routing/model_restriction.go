package routing

// ModelForRestriction 根据分组白名单检查阶段确定限制检查使用的模型。
// upstream 返回空（需逐提供商检查）。
func ModelForRestriction(source, requestedModel, groupMappedModel string) string {
	switch source {
	case BillingModelSourceRequested:
		return requestedModel
	case BillingModelSourceUpstream:
		return ""
	case BillingModelSourceGroupMapped:
		return groupMappedModel
	default:
		return groupMappedModel
	}
}
