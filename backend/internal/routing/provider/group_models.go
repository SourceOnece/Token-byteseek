package provider

import (
	"slices"

	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	claude "github.com/TokenFlux/TokenRouter/internal/upstream/anthropic"
	"github.com/TokenFlux/TokenRouter/internal/upstream/antigravity"
	"github.com/TokenFlux/TokenRouter/internal/upstream/deepseek"
	geminicli "github.com/TokenFlux/TokenRouter/internal/upstream/gemini/codeassist"
	xai "github.com/TokenFlux/TokenRouter/internal/upstream/grok"
	"github.com/TokenFlux/TokenRouter/internal/upstream/kimi"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
	"github.com/TokenFlux/TokenRouter/internal/upstream/zhipu"
)

// DefaultGroupModelCandidates 保留平台目录的原始顺序与动态读取时点。
func DefaultGroupModelCandidates(platform string) []string {
	if platform == "" {
		var models []string
		for _, source := range []string{routing.PlatformAnthropic, routing.PlatformOpenAI, routing.PlatformGemini, routing.PlatformAntigravity, routing.PlatformGrok, routing.PlatformQoder, routing.PlatformKimi, routing.PlatformZhipu, routing.PlatformDeepseek} {
			models = append(models, DefaultGroupModelCandidates(source)...)
		}
		slices.Sort(models)
		return slices.Compact(models)
	}

	switch platform {
	case capability.PlatformOpenAI:
		return openai.DefaultModelIDs()
	case capability.PlatformGemini:
		ids := make([]string, 0, len(geminicli.DefaultModels))
		for _, model := range geminicli.DefaultModels {
			ids = append(ids, model.ID)
		}
		return ids
	case capability.PlatformAntigravity:
		models := antigravity.DefaultModels()
		ids := make([]string, 0, len(models))
		for _, model := range models {
			ids = append(ids, model.ID)
		}
		return ids
	case capability.PlatformQoder:
		return qoder.DefaultRequestModelIDs()
	case capability.PlatformGrok:
		return xai.DefaultModelIDs()
	case capability.PlatformKimi:
		return []string{kimi.DefaultTestModel}
	case capability.PlatformZhipu:
		return []string{zhipu.DefaultTestModel}
	case capability.PlatformDeepseek:
		return []string{deepseek.DefaultTestModel, "deepseek-reasoner"}
	default:
		ids := make([]string, 0, len(claude.DefaultModels))
		for _, model := range claude.DefaultModels {
			ids = append(ids, model.ID)
		}
		return ids
	}
}
