package pricing

import (
	"encoding/json"
	"slices"
)

// LiteLLMModelPricing LiteLLM价格数据结构
// 只保留我们需要的字段，使用指针来处理可能缺失的值
type LiteLLMModelPricing struct {
	CacheCreation1hPricePresent bool                  `json:"-"`
	PriorityInputPresent        bool                  `json:"-"`
	PriorityOutputPresent       bool                  `json:"-"`
	PriorityCacheReadPresent    bool                  `json:"-"`
	PriorityCacheWritePresent   bool                  `json:"-"`
	Source                      string                `json:"source,omitempty"`
	PriceSources                map[string]string     `json:"price_sources,omitempty"`
	ContextPrices               []CatalogContextPrice `json:"context_prices,omitempty"`
	// 保留缓存桶是否显式存在，供默认价格页区分免费与不适用。
	CacheCreationPricePresent           bool    `json:"-"`
	CacheReadPricePresent               bool    `json:"-"`
	ImageInputPricePresent              bool    `json:"-"`
	ImageOutputPricePresent             bool    `json:"-"`
	InputCostPerToken                   float64 `json:"input_cost_per_token"`
	InputCostPerTokenPriority           float64 `json:"input_cost_per_token_priority"`
	OutputCostPerToken                  float64 `json:"output_cost_per_token"`
	OutputCostPerTokenPriority          float64 `json:"output_cost_per_token_priority"`
	CacheCreationInputTokenCost         float64 `json:"cache_creation_input_token_cost"`
	CacheCreationInputTokenCostPriority float64 `json:"cache_creation_input_token_cost_priority"`
	CacheCreationInputTokenCostAbove1hr float64 `json:"cache_creation_input_token_cost_above_1hr"`
	CacheReadInputTokenCost             float64 `json:"cache_read_input_token_cost"`
	CacheReadInputTokenCostPriority     float64 `json:"cache_read_input_token_cost_priority"`
	LongContextInputTokenThreshold      int     `json:"long_context_input_token_threshold,omitempty"`
	LongContextInputCostMultiplier      float64 `json:"long_context_input_cost_multiplier,omitempty"`
	LongContextOutputCostMultiplier     float64 `json:"long_context_output_cost_multiplier,omitempty"`
	SupportsServiceTier                 bool    `json:"supports_service_tier"`
	LiteLLMProvider                     string  `json:"litellm_provider"`
	Mode                                string  `json:"mode"`
	SupportsPromptCaching               bool    `json:"supports_prompt_caching"`
	OutputCostPerImage                  float64 `json:"output_cost_per_image"`       // 图片生成模型每张图片价格
	OutputCostPerImageToken             float64 `json:"output_cost_per_image_token"` // 图片输出 token 价格
	InputCostPerImageToken              float64 `json:"input_cost_per_image_token"`  // 图片输入 token 价格（如 gpt-image-2 图片编辑）
	CacheReadInputImageTokenCost        float64 `json:"cache_read_input_image_token_cost"`

	// 模型能力元数据：由模型广场下发给前端展示输入/输出模态，不参与计费。
	SupportedModalities       []string `json:"supported_modalities"`
	SupportedOutputModalities []string `json:"supported_output_modalities"`
	SupportsVision            bool     `json:"supports_vision"`
	SupportsAudioInput        bool     `json:"supports_audio_input"`
	SupportsAudioOutput       bool     `json:"supports_audio_output"`
	SupportsVideoInput        bool     `json:"supports_video_input"`

	// TokenPricingAbsent 表示源数据中 input/output token 价格均缺失（仅有图片价）。
	// 此类条目只可用于图片计费，token 计费必须回退到 fallback 或 fail-closed，
	// 否则 token 流量会被按 $0 计费。零值（false）表示条目具备 token 价格。
	TokenPricingAbsent bool `json:"-"`
}

// LiteLLMRawEntry 用于解析原始JSON数据
type LiteLLMRawEntry struct {
	Source                              string                `json:"source,omitempty"`
	PriceSources                        map[string]string     `json:"price_sources,omitempty"`
	ContextPrices                       []CatalogContextPrice `json:"context_prices,omitempty"`
	InputCostPerToken                   *float64              `json:"input_cost_per_token"`
	InputCostPerTokenPriority           *float64              `json:"input_cost_per_token_priority"`
	OutputCostPerToken                  *float64              `json:"output_cost_per_token"`
	OutputCostPerTokenPriority          *float64              `json:"output_cost_per_token_priority"`
	CacheCreationInputTokenCost         *float64              `json:"cache_creation_input_token_cost"`
	CacheCreationInputTokenCostPriority *float64              `json:"cache_creation_input_token_cost_priority"`
	CacheCreationInputTokenCostAbove1hr *float64              `json:"cache_creation_input_token_cost_above_1hr"`
	CacheReadInputTokenCost             *float64              `json:"cache_read_input_token_cost"`
	CacheReadInputTokenCostPriority     *float64              `json:"cache_read_input_token_cost_priority"`
	LongContextInputTokenThreshold      *int                  `json:"long_context_input_token_threshold"`
	LongContextInputCostMultiplier      *float64              `json:"long_context_input_cost_multiplier"`
	LongContextOutputCostMultiplier     *float64              `json:"long_context_output_cost_multiplier"`
	SupportsServiceTier                 bool                  `json:"supports_service_tier"`
	LiteLLMProvider                     string                `json:"litellm_provider"`
	Mode                                string                `json:"mode"`
	SupportsPromptCaching               bool                  `json:"supports_prompt_caching"`
	OutputCostPerImage                  *float64              `json:"output_cost_per_image"`
	OutputCostPerImageToken             *float64              `json:"output_cost_per_image_token"`
	InputCostPerImageToken              *float64              `json:"input_cost_per_image_token"`
	CacheReadInputImageTokenCost        *float64              `json:"cache_read_input_image_token_cost"`
	SupportedModalities                 []string              `json:"supported_modalities"`
	SupportedInputModalities            []string              `json:"supported_input_modalities"`
	SupportedOutputModalities           []string              `json:"supported_output_modalities"`
	SupportsVision                      bool                  `json:"supports_vision"`
	SupportsAudioInput                  bool                  `json:"supports_audio_input"`
	SupportsAudioOutput                 bool                  `json:"supports_audio_output"`
	SupportsVideoInput                  bool                  `json:"supports_video_input"`
}

// CatalogContextPrice 保存目录给出的绝对阶梯价，适用于超过 Threshold 的整次用量。
type CatalogContextPrice struct {
	Threshold int                  `json:"threshold"`
	Pricing   *LiteLLMModelPricing `json:"pricing"`
}

// CloneCatalogPrice 递归复制阶梯与来源标签，避免只读快照丢失零价存在性。
func CloneCatalogPrice(value *LiteLLMModelPricing) *LiteLLMModelPricing {
	if value == nil {
		return nil
	}
	result := *value
	result.SupportedModalities = slices.Clone(value.SupportedModalities)
	result.SupportedOutputModalities = slices.Clone(value.SupportedOutputModalities)
	if value.PriceSources != nil {
		result.PriceSources = make(map[string]string, len(value.PriceSources))
		for key, source := range value.PriceSources {
			result.PriceSources[key] = source
		}
	}
	if value.ContextPrices != nil {
		result.ContextPrices = make([]CatalogContextPrice, len(value.ContextPrices))
	}
	for i, tier := range value.ContextPrices {
		result.ContextPrices[i] = CatalogContextPrice{Threshold: tier.Threshold, Pricing: CloneCatalogPrice(tier.Pricing)}
	}
	return &result
}

// UnmarshalJSON 用同一价格解析器读取阶梯，保留缓存桶缺失与零价的区别。
func (c *CatalogContextPrice) UnmarshalJSON(body []byte) error {
	var raw struct {
		Threshold int             `json:"threshold"`
		Pricing   json.RawMessage `json:"pricing"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	entries, _, err := ParsePricingEntries(map[string]json.RawMessage{"tier": raw.Pricing})
	if err != nil {
		return err
	}
	c.Threshold, c.Pricing = raw.Threshold, entries["tier"]
	return nil
}
