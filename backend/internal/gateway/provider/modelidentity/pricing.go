package modelidentity

import (
	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"
)

// CandidatesFactory 返回统一的完整模型身份查询。
func CandidatesFactory() func(string) []string {
	return LookupCandidates
}

// LookupCandidates 保留完整型号，仅解析协议资源路径。
func LookupCandidates(model string) []string {
	return pricing.BuildModelIdentityCandidates(model)
}

// Identity 投影一次模型查询所需的候选与规范名称。
func Identity(model string) billing.ModelIdentity {
	return billing.ModelIdentity{Candidates: LookupCandidates(model)}
}

// PricingPolicy 把平台型号身份投影给唯一的纯定价规则。
func PricingPolicy(model string) pricing.ModelPolicy {
	normalized := NormalizeOpenAI(model)
	return pricing.ModelPolicy{
		NormalizedOpenAIModel: normalized,
		IsGPT56:               IsGPT56(normalized),
	}
}
