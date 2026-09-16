package handler

import "github.com/TokenFlux/TokenRouter/internal/service"

// 全部候选都要命中，避免重复 model 键被不同解析器选成不同值。
func blockedModelAllowlistCandidate(group *service.Group, candidates []string) string {
	if group == nil || !group.ModelAllowlistEnabled() {
		return ""
	}
	for _, model := range candidates {
		if !group.ModelAllowlist.Allows(model) {
			return model
		}
	}
	return ""
}
