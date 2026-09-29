package provider

import (
	"slices"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/upstream/bedrock"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/anthropic"
	"github.com/TokenFlux/TokenRouter/internal/upstream/antigravity"
	"github.com/TokenFlux/TokenRouter/internal/upstream/deepseek"
	"github.com/TokenFlux/TokenRouter/internal/upstream/gemini"
	"github.com/TokenFlux/TokenRouter/internal/upstream/gemini/codeassist"
	"github.com/TokenFlux/TokenRouter/internal/upstream/grok"
	"github.com/TokenFlux/TokenRouter/internal/upstream/kimi"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
	"github.com/TokenFlux/TokenRouter/internal/upstream/zhipu"
)

// DefaultProviderModels 复用各上游拥有的目录，未知平台没有隐式全模型能力。
func DefaultProviderModels(value *provider.Record) []string {
	if value == nil {
		return nil
	}
	switch value.Platform {
	case provider.PlatformAnthropic:
		if value.Type == provider.ProviderTypeBedrock {
			var models []string
			for alias, target := range bedrock.DefaultBedrockModelMapping {
				models = append(models, alias, target)
			}
			slices.Sort(models)
			return slices.Compact(models)
		}
		return anthropic.DefaultModelIDs()
	case provider.PlatformOpenAI:
		if value.IsShadow() {
			return sparkModelVariants()
		}
		models := openai.DefaultModelIDs()
		if value.Type == provider.ProviderTypeAPIKey {
			models = append(models, "text-embedding-3-small", "text-embedding-3-large", "text-embedding-ada-002")
		}
		return models
	case provider.PlatformGemini:
		var models []string
		if value.IsGeminiGoogleOne() {
			for _, model := range codeassist.GoogleOneModels {
				models = append(models, model.ID)
			}
		} else if value.Type == provider.ProviderTypeOAuth {
			for _, model := range codeassist.DefaultModels {
				models = append(models, model.ID)
			}
		} else {
			for _, model := range gemini.DefaultModels() {
				models = append(models, strings.TrimPrefix(model.Name, "models/"))
			}
		}
		return models
	case provider.PlatformAntigravity:
		var models []string
		for _, model := range antigravity.DefaultModels() {
			models = append(models, model.ID)
		}
		return models
	case provider.PlatformGrok:
		return grok.DefaultModelIDs()
	case provider.PlatformQoder:
		site, err := qoder.ParseSite(value.GetCredential("site"))
		if err != nil {
			return nil
		}
		return qoder.DefaultRequestModelIDsForSite(site)
	case provider.PlatformKimi:
		return []string{kimi.DefaultTestModel}
	case provider.PlatformZhipu:
		return []string{zhipu.DefaultTestModel}
	case provider.PlatformDeepseek:
		return []string{deepseek.DefaultTestModel, "deepseek-reasoner"}
	case provider.PlatformMiniMax:
		return provider.MiniMaxDefaultModelIDs()
	case provider.PlatformOpenCodeGo:
		return provider.DefaultOpenCodeGoModelIDs()
	default:
		return nil
	}
}
