package pricing

import (
	"github.com/TokenFlux/TokenRouter/internal/protocol/openai"
)

// 档位标识复用协议值，计价策略由当前包拥有。
const (
	OpenAIFastTierPriority  = openai.ServiceTierPriority
	OpenAIFastTierUltrafast = openai.ServiceTierUltrafast
)
