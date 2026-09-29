package routing

import (
	"sort"
	"strings"
)

// ModelRejectionRules 只读取候选资格与显式模型规则，不读取提供商凭据或存储。
type ModelRejectionRules interface {
	IsSchedulable() bool
	GetConfiguredRequestModels() []string
	IsModelSupported(string) bool
}

// ModelRejectionSource 保留逐提供商懒读取，默认目录由平台适配器提供。
type ModelRejectionSource struct {
	Platform string
	Rules    ModelRejectionRules
	Defaults func(string) ([]string, error)
}

// AvailableModelsForRejection 只生成原错误展示目录，不扩展可调度能力。
func AvailableModelsForRejection(providers []ModelRejectionSource, platform string) []string {
	modelSet := make(map[string]struct{})
	hasConfiguredModels := false
	for i := range providers {
		value := &providers[i]
		if !value.Rules.IsSchedulable() || !matchesRejectionPlatform(value, platform) {
			continue
		}
		requestModels := value.Rules.GetConfiguredRequestModels()
		if len(requestModels) == 0 {
			if value.Defaults == nil {
				continue
			}
			defaultModels, err := value.Defaults(value.Platform)
			if err != nil {
				continue
			}
			for _, model := range defaultModels {
				if model = strings.TrimSpace(model); model != "" && !strings.Contains(model, "*") && value.Rules.IsModelSupported(model) {
					modelSet[model] = struct{}{}
				}
			}
			continue
		}
		hasConfiguredModels = true
		for _, model := range requestModels {
			if model = strings.TrimSpace(model); model != "" && !strings.Contains(model, "*") && value.Rules.IsModelSupported(model) {
				modelSet[model] = struct{}{}
			}
		}
	}
	if len(modelSet) == 0 && !hasConfiguredModels {
		return nil
	}
	models := make([]string, 0, len(modelSet))
	for model := range modelSet {
		models = append(models, model)
	}
	sort.Strings(models)
	return models
}

func matchesRejectionPlatform(value *ModelRejectionSource, platform string) bool {
	return platform == "" || value.Platform == platform
}

// NewGroupModelRejection 保留空请求或空候选时不新增错误的原边界。
func NewGroupModelRejection(platform, requested string, providers []ModelRejectionSource) error {
	requested = strings.TrimSpace(requested)
	if requested == "" || len(providers) == 0 {
		return nil
	}
	return &GroupModelUnsupportedError{Platform: platform, RequestedModel: requested, AvailableModels: AvailableModelsForRejection(providers, platform)}
}
