package selection

import (
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
)

// modelRejectionSources 只投影旧候选的原读取端口；过滤、聚合及错误归 routing。
func modelRejectionSources(providers []gatewayprovider.ExecutionProvider) []routing.ModelRejectionSource {
	sources := make([]routing.ModelRejectionSource, len(providers))
	for i := range providers {
		value := &providers[i]
		sources[i] = gatewayprovider.ModelRejectionProvider(gatewayprovider.ExecutionRecord(value))
	}
	return sources
}
