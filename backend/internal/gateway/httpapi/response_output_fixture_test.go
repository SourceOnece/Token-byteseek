package httpapi

import (
	"time"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// newResponseOutputForTest 只构造响应组件的真实依赖，不创建旧网关、选号或完成队列。
func newResponseOutputForTest(options OpenAIResponseOptions) *OpenAIResponseOutput {
	if options.ReadLimit == 0 {
		options.ReadLimit = 64 << 20
	}
	blocks := provider.NewRuntimeBlockState(time.Now)
	models := provider.NewModelTransientState(0)
	return &OpenAIResponseOutput{
		Options: options,
		Health:  &provideradapter.OpenAIResponseHealth{Runtime: blocks, ModelTransient: models},
		GrokHealth: &provideradapter.GrokHealth{Runtime: blocks, ModelTransient: models, NormalizeModel: func(value *provider.Record, model string) string {
			return (gatewayadapter.ModelPolicy{Record: value}).NormalizeOpenAI(model)
		}},
		Turns:        &CodexTurnStateHeaders{Origins: session.NewCodexTurnOrigins(time.Now), TTL: func() time.Duration { return time.Hour }},
		ProxyCircuit: egress.NewProxyStreamCircuit(egress.DefaultProxyStreamCircuitSettings()),
		Reasoning:    &session.ReasoningHistory{Warn: gatewayadapter.WarnReasoningCacheFailure},
		ResponseTTL:  func() time.Duration { return time.Hour },
	}
}
