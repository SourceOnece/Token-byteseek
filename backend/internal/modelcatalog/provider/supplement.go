package provider

import "github.com/TokenFlux/TokenRouter/internal/billing/pricing"

// supplementFields 限定本地补充可填的计费维度，不向展示属性写入旧能力字段。
var supplementFields = map[string]string{
	"input_cost_per_token": "input", "output_cost_per_token": "output",
	"input_cost_per_token_priority": "priority_input", "output_cost_per_token_priority": "priority_output",
	"cache_read_input_token_cost": "cache_read", "cache_read_input_token_cost_priority": "priority_cache_read",
	"cache_creation_input_token_cost": "cache_write", "cache_creation_input_token_cost_priority": "priority_cache_write",
	"cache_creation_input_token_cost_above_1hr": "cache_write_1h",
	"input_cost_per_image_token":                "image_input", "output_cost_per_image_token": "image_output", "output_cost_per_image": "image",
	"image_prices": "image_prices", "video_prices": "video_prices",
	"fast_multiplier": "fast_multiplier", "flex_multiplier": "flex_multiplier",
	"max_reasoning_effort_multiplier":     "max_reasoning_effort_multiplier",
	"ultrafast_multiplier":                "ultrafast_multiplier",
	"cache_read_input_image_token_cost":   "image_cache_read",
	"long_context_input_token_threshold":  "long_context_threshold",
	"long_context_input_cost_multiplier":  "long_context_input",
	"long_context_output_cost_multiplier": "long_context_output",
	"cache_write_multiplier":              "cache_write_multiplier", "cache_write_1h_multiplier": "cache_write_1h_multiplier",
	"time_pricing": "time_pricing", "source_url": "source_url", "verified_at": "verified_at",
}

// BillingDefaults 返回当前目录版本的操作价格，不将保留节点注册为模型。
func (s *Service) BillingDefaults() pricing.OperationPrices {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.billingDefaults.Clone()
}
