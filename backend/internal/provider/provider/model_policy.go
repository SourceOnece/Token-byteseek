package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/antigravity"
	"github.com/TokenFlux/TokenRouter/internal/upstream/gemini/codeassist"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
)

// ModelDefaults 提供原平台目录的懒读取入口，不复制映射或建立新缓存。
func ModelDefaults() provider.ModelMappingDefaults {
	return provider.ModelMappingDefaults{
		Models:                DefaultProviderModels,
		Antigravity:           func() map[string]string { return antigravity.DefaultAntigravityModelMapping },
		GoogleOne:             codeassist.GoogleOneModelMapping,
		AntigravityAgentModel: antigravity.AntigravityGemini31ProAgentModel,
	}
}

// ModelRules 仅在核心需要平台资格时读取站点；不提前解析或扩大提供商原生能力。
func ModelRules(value *provider.Record) provider.ModelPlatformRules {
	return provider.ModelPlatformRules{
		NormalizeQoder:      qoder.NormalizeModelForWhitelist,
		NormalizeOpenAI:     openai.GetNormalizedCodexModel,
		OpenAIOAuthServable: provider.IsOpenAIOAuthServableModel,
		QoderCompatible: func(model string) bool {
			if value == nil {
				return false
			}
			site, err := qoder.ParseSite(value.GetCredential("site"))
			return err == nil && qoder.ModelCompatibleWithSite(site, model)
		},
	}
}

// SupportsOpenAIEndpoint 在端点能力检查实际需要时提供平台媒体资格，不提前读取资格。
func SupportsOpenAIEndpoint(value *provider.Record, capability provider.OpenAIEndpointCapability) bool {
	return value.SupportsOpenAIEndpointCapability(capability, func() (bool, string) {
		return provider.GrokMediaGenerationEligibility(value, GrokTierRules())
	})
}
