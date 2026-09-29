package app

import (
	"time"

	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// provideGrokHealth 与 OpenAI 快照写入共享节流器，与选号共享运行阻断及模型冷却。
func provideGrokHealth(store *providerpostgres.ProviderStore, health *provideradapter.UpstreamHealth, blocks *provider.RuntimeBlockState, models *provider.ModelTransientState) *provideradapter.GrokHealth {
	return &provideradapter.GrokHealth{
		Store: store, Health: health, Runtime: blocks, ModelTransient: models,
		Throttle: provider.NewWriteThrottle(30 * time.Second),
		NormalizeModel: func(value *provider.Record, model string) string {
			return (gatewayadapter.ModelPolicy{Record: value}).NormalizeOpenAI(model)
		},
	}
}
