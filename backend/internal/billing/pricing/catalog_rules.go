package pricing

import (
	"bytes"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	// AboveTierPricePattern 匹配目录中的长上下文绝对价字段。
	// 服务档后缀和 cache 侧字段不参与阈值及倍率折算。
	AboveTierPricePattern = regexp.MustCompile(`^(input|output)_cost_per_token_above_(\d+)k_tokens$`)
	// CacheTierPricePattern 匹配 cache 侧长上下文绝对价字段，用于数据契约告警。
	// 组 1 为缓存基础价字段，组 2 为 1 小时缓存时长段，组 3 为服务档后缀。
	CacheTierPricePattern       = regexp.MustCompile(`^(cache_(?:creation|read)_input_token_cost)(_above_1hr)?_above_\d+k_tokens((?:_[a-z]+)?)$`)
	ClaudeOpus48FallbackPricing = &CatalogModelPricing{
		InputCostPerToken:                   5e-06,  // 每百万 token $5
		OutputCostPerToken:                  25e-06, // 每百万 token $25
		CacheCreationInputTokenCost:         6.25e-06,
		CacheCreationInputTokenCostAbove1hr: 10e-06,
		CacheReadInputTokenCost:             0.5e-06,
		Provider:                            "anthropic",
		Mode:                                "chat",
		SupportsPromptCaching:               true,
		// Claude Opus 4.8 Fast mode 官方价格是常规定价的 2 倍，复用通用 service_tier 倍率即可。
		SupportsServiceTier: true,
	}
	OpenAIGPT55FallbackPricing = &CatalogModelPricing{
		InputCostPerToken:               5e-06,    // $5 per MTok
		InputCostPerTokenPriority:       12.5e-06, // $12.5 per MTok
		OutputCostPerToken:              3e-05,    // $30 per MTok
		OutputCostPerTokenPriority:      7.5e-05,  // $75 per MTok
		CacheCreationInputTokenCost:     5e-06,    // $5 per MTok
		CacheReadInputTokenCost:         5e-07,    // $0.5 per MTok
		CacheReadInputTokenCostPriority: 1.25e-06, // $1.25 per MTok
		SupportsServiceTier:             true,
		Provider:                        "openai",
		Mode:                            "chat",
		SupportsPromptCaching:           true,
	}
	// GPT-6 Astra 静态回退只固化官方标准价，避免目录缺失时误落到旧型号。
	OpenAIGPT6AstraPricing = &CatalogModelPricing{
		InputCostPerToken:           10e-6,   // 每百万 token $10
		OutputCostPerToken:          50e-6,   // 每百万 token $50
		CacheCreationInputTokenCost: 12.5e-6, // 每百万 token $12.50
		CacheReadInputTokenCost:     1e-6,    // 每百万 token $1
		SupportsServiceTier:         true,
		Provider:                    "openai",
		Mode:                        "chat",
		SupportsPromptCaching:       true,
	}
	// 与 sub2api 0.2.11 的 Sol 标准、Fast 和长上下文报价一致。
	OpenAIGPT61SolPricing = &CatalogModelPricing{
		InputCostPerToken: 2e-6, InputCostPerTokenPriority: 4e-6,
		OutputCostPerToken: 10e-6, OutputCostPerTokenPriority: 20e-6,
		CacheCreationInputTokenCost: 2.5e-6, CacheCreationInputTokenCostPriority: 5e-6,
		CacheReadInputTokenCost: .1e-6, CacheReadInputTokenCostPriority: .2e-6,
		LongContextInputTokenThreshold: 272000, LongContextInputCostMultiplier: 2, LongContextOutputCostMultiplier: 1.5,
		SupportsServiceTier: true, Provider: "openai", Mode: "chat", SupportsPromptCaching: true,
	}
	OpenAIGPT56SolPricing = &CatalogModelPricing{
		InputCostPerToken:                   5e-06,   // $5 per MTok
		InputCostPerTokenPriority:           1e-05,   // $10 per MTok
		OutputCostPerToken:                  3e-05,   // $30 per MTok
		OutputCostPerTokenPriority:          6e-05,   // $60 per MTok
		CacheCreationInputTokenCost:         6.25e-6, // $6.25 per MTok
		CacheCreationInputTokenCostPriority: 1.25e-5, // $12.5 per MTok
		CacheReadInputTokenCost:             5e-07,   // $0.50 per MTok
		CacheReadInputTokenCostPriority:     1e-06,   // $1 per MTok
		SupportsServiceTier:                 true,
		Provider:                            "openai",
		Mode:                                "chat",
		SupportsPromptCaching:               true,
	}
	OpenAIGPT56TerraPricing = &CatalogModelPricing{
		InputCostPerToken:                   2e-06,   // 每百万 token $2
		InputCostPerTokenPriority:           4e-06,   // 每百万 token $4
		OutputCostPerToken:                  1.2e-05, // 每百万 token $12
		OutputCostPerTokenPriority:          2.4e-05, // 每百万 token $24
		CacheCreationInputTokenCost:         2.5e-6,  // 每百万 token $2.50
		CacheCreationInputTokenCostPriority: 5e-6,    // 每百万 token $5
		CacheReadInputTokenCost:             2e-07,   // 每百万 token $0.20
		CacheReadInputTokenCostPriority:     4e-07,   // 每百万 token $0.40
		SupportsServiceTier:                 true,
		Provider:                            "openai",
		Mode:                                "chat",
		SupportsPromptCaching:               true,
	}
	OpenAIGPT56LunaPricing = &CatalogModelPricing{
		InputCostPerToken:                   2e-07,   // 每百万 token $0.20
		InputCostPerTokenPriority:           4e-07,   // 每百万 token $0.40
		OutputCostPerToken:                  1.2e-06, // 每百万 token $1.20
		OutputCostPerTokenPriority:          2.4e-06, // 每百万 token $2.40
		CacheCreationInputTokenCost:         2.5e-7,  // 每百万 token $0.25
		CacheCreationInputTokenCostPriority: 5e-7,    // 每百万 token $0.50
		CacheReadInputTokenCost:             2e-08,   // 每百万 token $0.02
		CacheReadInputTokenCostPriority:     4e-08,   // 每百万 token $0.04
		SupportsServiceTier:                 true,
		Provider:                            "openai",
		Mode:                                "chat",
		SupportsPromptCaching:               true,
	}
	OpenAIGPT55ProFallbackPricing = &CatalogModelPricing{
		InputCostPerToken:               3e-05,   // $30 per MTok
		InputCostPerTokenPriority:       7.5e-05, // $75 per MTok
		OutputCostPerToken:              1.8e-04, // $180 per MTok
		OutputCostPerTokenPriority:      4.5e-04, // $450 per MTok
		CacheCreationInputTokenCost:     3e-05,   // $30 per MTok
		CacheReadInputTokenCost:         3e-06,   // $3 per MTok
		CacheReadInputTokenCostPriority: 7.5e-06, // $7.5 per MTok
		SupportsServiceTier:             true,
		Provider:                        "openai",
		Mode:                            "responses",
		SupportsPromptCaching:           true,
	}
	OpenAIGPT54FallbackPricing = &CatalogModelPricing{
		InputCostPerToken:       2.5e-06, // $2.5 per MTok
		OutputCostPerToken:      1.5e-05, // $15 per MTok
		CacheReadInputTokenCost: 2.5e-07, // $0.25 per MTok
		Provider:                "openai",
		Mode:                    "chat",
		SupportsPromptCaching:   true,
	}
	OpenAIGPT54MiniFallbackPricing = &CatalogModelPricing{
		InputCostPerToken:       7.5e-07,
		OutputCostPerToken:      4.5e-06,
		CacheReadInputTokenCost: 7.5e-08,
		Provider:                "openai",
		Mode:                    "chat",
		SupportsPromptCaching:   true,
	}
	OpenAIGPT54NanoFallbackPricing = &CatalogModelPricing{
		InputCostPerToken:       2e-07,
		OutputCostPerToken:      1.25e-06,
		CacheReadInputTokenCost: 2e-08,
		Provider:                "openai",
		Mode:                    "chat",
		SupportsPromptCaching:   true,
	}
)

// DeriveLongContextFromAboveTierFields 将目录中的 above_XXXk 绝对价折算为本 fork
// 计费模型使用的阈值和倍率。多个阈值同时存在时取最小阈值；cache 侧 above 价由
// 计费核心按输入倍率统一处理，不在此处单独写入结构体。
func DeriveLongContextFromAboveTierFields(rawEntry json.RawMessage, pricing *CatalogModelPricing) {
	if pricing == nil ||
		pricing.LongContextInputTokenThreshold > 0 ||
		pricing.LongContextInputCostMultiplier > 0 ||
		pricing.LongContextOutputCostMultiplier > 0 {
		return
	}
	if !bytes.Contains(rawEntry, []byte("_above_")) {
		return
	}
	var fields map[string]any
	if err := json.Unmarshal(rawEntry, &fields); err != nil {
		return
	}
	type tierPrices struct{ input, output float64 }
	tiers := make(map[int]*tierPrices)
	for key, value := range fields {
		match := AboveTierPricePattern.FindStringSubmatch(key)
		if match == nil {
			continue
		}
		price, ok := value.(float64)
		if !ok || price <= 0 {
			continue
		}
		thousands, err := strconv.Atoi(match[2])
		if err != nil || thousands <= 0 {
			continue
		}
		threshold := thousands * 1000
		tier := tiers[threshold]
		if tier == nil {
			tier = &tierPrices{}
			tiers[threshold] = tier
		}
		if match[1] == "input" {
			tier.input = price
		} else {
			tier.output = price
		}
	}
	if len(tiers) == 0 {
		return
	}
	threshold := 0
	for candidate := range tiers {
		if threshold == 0 || candidate < threshold {
			threshold = candidate
		}
	}
	tier := tiers[threshold]
	inputMultiplier, outputMultiplier := 1.0, 1.0
	if tier.input > 0 && pricing.InputCostPerToken > 0 {
		inputMultiplier = tier.input / pricing.InputCostPerToken
	}
	if tier.output > 0 && pricing.OutputCostPerToken > 0 {
		outputMultiplier = tier.output / pricing.OutputCostPerToken
	}
	// above 价格没有高于基础价时不创建阶梯，避免错误目录导致降价。
	if inputMultiplier <= 1 && outputMultiplier <= 1 {
		return
	}
	pricing.LongContextInputTokenThreshold = threshold
	pricing.LongContextInputCostMultiplier = inputMultiplier
	pricing.LongContextOutputCostMultiplier = outputMultiplier
}

// IsLopsidedLongContextLadder 判断折算后的阶梯是否只有输入或输出一侧有附加费。
func IsLopsidedLongContextLadder(pricing *CatalogModelPricing) bool {
	if pricing == nil || pricing.LongContextInputTokenThreshold <= 0 {
		return false
	}
	return (pricing.LongContextInputCostMultiplier > 1) != (pricing.LongContextOutputCostMultiplier > 1)
}

// OrphanCacheTierFields 找出没有可回落基础价的 cache above 字段，供加载时告警。
func OrphanCacheTierFields(rawEntry json.RawMessage) []string {
	if !bytes.Contains(rawEntry, []byte("_above_")) {
		return nil
	}
	var fields map[string]any
	if err := json.Unmarshal(rawEntry, &fields); err != nil {
		return nil
	}
	positive := func(key string) bool {
		price, ok := fields[key].(float64)
		return ok && price > 0
	}
	var orphans []string
	for key := range fields {
		match := CacheTierPricePattern.FindStringSubmatch(key)
		if match == nil || !positive(key) {
			continue
		}
		stem, hourly, tier := match[1], match[2], match[3]
		if positive(stem+hourly+tier) || positive(stem+hourly) || positive(stem+tier) || positive(stem) {
			continue
		}
		orphans = append(orphans, key)
	}
	sort.Strings(orphans)
	return orphans
}

// MergePricingOverrideEntry 在 JSON 字段层浅合并：patch 字段覆盖 base 同名字段，
// 值为 null 的 patch 字段从结果中删除，base 为空时结果即 patch 本身。
// patch 不是 JSON 对象时返回 ok=false。
func MergePricingOverrideEntry(base, patch json.RawMessage) (json.RawMessage, bool) {
	var patchFields map[string]any
	if err := json.Unmarshal(patch, &patchFields); err != nil || patchFields == nil {
		return nil, false
	}
	merged := make(map[string]any, len(patchFields))
	if len(base) > 0 {
		// base 非对象时忽略，仅以 patch 为准。
		if err := json.Unmarshal(base, &merged); err != nil {
			merged = make(map[string]any, len(patchFields))
		}
	}
	for k, v := range patchFields {
		if v == nil {
			delete(merged, k)
			continue
		}
		merged[k] = v
	}
	out, err := json.Marshal(merged)
	if err != nil {
		return nil, false
	}
	return out, true
}

// 模型广场可下发的模态取值白名单与固定输出顺序。
var MarketplaceModalityOrder = []string{"text", "image", "audio", "video"}

// SanitizeModalities 过滤定价文件中的非模态取值并去重，按固定顺序输出。
func SanitizeModalities(values []string) []string {
	present := make(map[string]bool, len(values))
	for _, value := range values {
		present[strings.ToLower(strings.TrimSpace(value))] = true
	}
	out := make([]string, 0, len(MarketplaceModalityOrder))
	for _, modality := range MarketplaceModalityOrder {
		if present[modality] {
			out = append(out, modality)
		}
	}
	return out
}

// DeriveModalities 从定价条目合成输入/输出模态：supported_modalities 缺失的一侧
// 用 mode 兜底，再用 supports_* 标记和图片输入价补充（图片编辑体现为图片输入价）。
func DeriveModalities(p *CatalogModelPricing) ([]string, []string) {
	if p == nil {
		return nil, nil
	}

	// 复制后再追加，避免并发查询时写共享底层数组（pricingData 里的切片被多个请求复用）。
	input := append([]string{}, p.SupportedModalities...)
	output := append([]string{}, p.SupportedOutputModalities...)
	if len(input) == 0 || len(output) == 0 {
		modeInput, modeOutput := ModalitiesFromMode(p.Mode)
		if len(input) == 0 {
			input = modeInput
		}
		if len(output) == 0 {
			output = modeOutput
		}
	}
	if p.SupportsVision {
		input = append(input, "image")
	}
	if p.SupportsAudioInput {
		input = append(input, "audio")
	}
	if p.SupportsAudioOutput {
		output = append(output, "audio")
	}
	if p.SupportsVideoInput {
		input = append(input, "video")
	}
	if p.InputCostPerImageToken > 0 {
		input = append(input, "image")
	}

	in := SanitizeModalities(input)
	out := SanitizeModalities(output)
	if len(in) == 0 || len(out) == 0 {
		return nil, nil
	}
	return in, out
}

// ModalitiesFromMode 按 模型目录 mode 推断基础模态；未知 mode 一律按文字模型处理。
func ModalitiesFromMode(mode string) ([]string, []string) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "image_generation":
		return []string{"text"}, []string{"image"}
	case "audio_transcription":
		return []string{"audio"}, []string{"text"}
	case "audio_speech":
		return []string{"text"}, []string{"audio"}
	case "realtime":
		return []string{"text", "audio"}, []string{"text", "audio"}
	default:
		// chat/responses/completion 等对话类模式至少支持文字输入输出。
		return []string{"text"}, []string{"text"}
	}
}

// BuildModelLookupCandidates 为价格和属性提供相同的完整模型身份。
// @project-doc docs/interfaces/model_catalog_and_marketplace.md#model_catalog_metadata_lookup
func BuildModelLookupCandidates(model string) []string {
	return BuildModelIdentityCandidates(model)
}

// BuildModelIdentityCandidates 仅解析已知 Google 资源路径，保留供应商限定名和版本。
func BuildModelIdentityCandidates(model string) []string {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return nil
	}
	normalized := NormalizeModelNameForPricing(model)
	if normalized == model {
		return []string{model}
	}
	return []string{model, normalized}
}

func NormalizeModelNameForPricing(model string) string {
	// 这里只清理资源路径和名称写法，不移除档位或改成其它产品。
	model = strings.TrimSpace(model)
	model = strings.TrimLeft(model, "/")
	model = strings.TrimPrefix(model, "models/")
	model = strings.TrimPrefix(model, "publishers/google/models/")

	if idx := strings.LastIndex(model, "/publishers/google/models/"); idx != -1 {
		model = model[idx+len("/publishers/google/models/"):]
	}

	model = strings.TrimLeft(model, "/")
	return model
}
