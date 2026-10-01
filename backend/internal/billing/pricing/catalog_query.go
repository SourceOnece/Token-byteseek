package pricing

import (
	"strings"
)

// GetModelPricing 按完整模型身份查询目录，不借用其它型号的价格。
func (s *CatalogQuery) GetModelPricing(modelName string) *CatalogModelPricing {
	candidates := s.modelLookupCandidates(modelName)
	if entry := s.LookupModelCatalogEntry(candidates); entry != nil {
		return entry
	}
	for _, model := range candidates {
		switch model {
		case "claude-opus-4-8":
			return ClaudeOpus48FallbackPricing
		case "gpt-5.4":
			return OpenAIGPT54FallbackPricing
		case "gpt-5.4-mini":
			return OpenAIGPT54MiniFallbackPricing
		case "gpt-5.4-nano":
			return OpenAIGPT54NanoFallbackPricing
		case "gpt-5.5":
			return OpenAIGPT55FallbackPricing
		case "gpt-5.5-pro":
			return OpenAIGPT55ProFallbackPricing
		case "gpt-5.6-sol":
			return OpenAIGPT56SolPricing
		case "gpt-5.6-terra":
			return OpenAIGPT56TerraPricing
		case "gpt-5.6-luna":
			return OpenAIGPT56LunaPricing
		case "gpt-6.1-sol":
			return OpenAIGPT61SolPricing
		case "gpt-6-astra":
			return OpenAIGPT6AstraPricing
		}
	}
	return nil
}

// GetModelModalities 只读取完整目录条目的模态，不借用静态价格或其它型号的能力。
func (s *CatalogQuery) GetModelModalities(modelName string) ([]string, []string) {
	if s == nil {
		return nil, nil
	}

	modelLower := strings.ToLower(strings.TrimSpace(modelName))
	if modelLower == "" {
		return nil, nil
	}

	lookupCandidates := s.modelLookupCandidates(modelLower)
	return DeriveModalities(s.LookupModelCatalogEntry(lookupCandidates))
}

// LookupModelCatalogEntry 按候选顺序查询显式目录；调用期间目录保持只读。
func (s *CatalogQuery) LookupModelCatalogEntry(candidates []string) *CatalogModelPricing {
	for _, candidate := range candidates {
		if pricing := s.Entries[candidate]; pricing != nil {
			return pricing
		}
	}
	return nil
}

// CatalogQuery 使用同一份完整模型身份查询价格与属性。
type CatalogQuery struct {
	Entries    map[string]*CatalogModelPricing
	Candidates func(string) []string
}

func (s *CatalogQuery) modelLookupCandidates(model string) []string {
	if s.Candidates != nil {
		return s.Candidates(model)
	}
	return BuildModelLookupCandidates(model)
}
