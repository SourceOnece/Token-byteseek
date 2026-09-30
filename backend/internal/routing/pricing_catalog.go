package routing

import (
	"fmt"
	"slices"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"

	infraerrors "github.com/TokenFlux/TokenRouter/internal/pkg/apperror"
)

type ModelPriceReader interface {
	GetModelPricing(string) (*ModelPricing, error)
}

// PricingCatalog 只组合现有价格目录和平台默认列表，不构建第二份目录缓存。
type PricingCatalog struct {
	Snapshot        func() DefaultPricingSnapshot
	Update          func() error
	Prices          ModelPriceReader
	NamesByProvider func(string) []string
	QoderModels     func() []string
}

func (c *PricingCatalog) DefaultPricing(model string) (*ModelPricing, error) {
	return c.Prices.GetModelPricing(model)
}

func (c *PricingCatalog) ModelNames(platform string) ([]string, error) {
	if platform == "" {
		var models []string
		if c.Snapshot != nil {
			for _, entry := range c.Snapshot().Prices {
				models = append(models, entry.Model)
			}
		} else if c.NamesByProvider != nil {
			for _, provider := range platformToLiteLLMProvider {
				models = append(models, c.NamesByProvider(provider)...)
			}
		}
		if c.QoderModels != nil {
			models = append(models, c.QoderModels()...)
		}
		slices.Sort(models)
		return slices.Compact(models), nil
	}

	if platform == PlatformQoder {
		return c.QoderModels(), nil
	}
	provider, ok := platformToLiteLLMProvider[platform]
	if !ok {
		return nil, infraerrors.BadRequest("UNSUPPORTED_PLATFORM", fmt.Sprintf("unsupported platform: %s", platform)).WithMetadata(map[string]string{"param": platform})
	}
	return c.NamesByProvider(provider), nil
}

// platformToLiteLLMProvider 将网关平台名映射为 LiteLLM 定价目录中的 provider
var platformToLiteLLMProvider = map[string]string{
	PlatformAnthropic:   "anthropic",
	PlatformOpenAI:      "openai",
	PlatformGemini:      "gemini",
	PlatformAntigravity: "anthropic",
	PlatformGrok:        "xai",
	PlatformKimi:        "moonshot",
	PlatformZhipu:       "zhipu",
	PlatformDeepseek:    "deepseek",
}

// DefaultPricingSnapshot 固定一次查询的模型名、价格和更新时间。
type DefaultPricingSnapshot struct {
	Version   string
	LastError string
	Prices    []pricing.DefaultModelPrice
	UpdatedAt time.Time
}
