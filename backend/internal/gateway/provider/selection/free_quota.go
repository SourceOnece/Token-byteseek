package selection

import (
	"context"
	"sync/atomic"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
)

func (s *compatiblePicker) filterGrokFreeQuotaProviders(_ context.Context, providers []gatewayprovider.ExecutionProvider) []gatewayprovider.ExecutionProvider {
	if s == nil || s.service == nil || s.service.newFreeQuotaGate == nil {
		return providers
	}
	gate := loadFreeQuotaGate(&s.freeQuotaGate, s.service.newFreeQuotaGate)
	return filterFreeQuotaProjection(gate, providers)
}

// loadFreeQuotaGate 只登记一个实例，构造未启动工作，因此竞争中未发布对象不需要清理。
func loadFreeQuotaGate(slot *atomic.Pointer[provider.FreeQuotaGate], factory func() *provider.FreeQuotaGate) *provider.FreeQuotaGate {
	if gate := slot.Load(); gate != nil {
		return gate
	}
	gate := factory()
	if slot.CompareAndSwap(nil, gate) {
		return gate
	}
	return slot.Load()
}

func (s *Generic) filterGrokFreeQuotaProvidersForGateway(_ context.Context, providers []gatewayprovider.ExecutionProvider) []gatewayprovider.ExecutionProvider {
	if s == nil {
		return providers
	}
	return filterFreeQuotaProjection(s.freeQuotaGate, providers)
}

// 旧执行形状仅投影资格与结果，不拥有缓存或裁决算法。
func filterFreeQuotaProjection(gate *provider.FreeQuotaGate, providers []gatewayprovider.ExecutionProvider) []gatewayprovider.ExecutionProvider {
	if gate == nil {
		return providers
	}
	candidates := make([]provider.FreeQuotaCandidate, len(providers))
	for i := range providers {
		candidates[i] = provider.FreeQuotaCandidate{ID: providers[i].Record.ID, Eligible: provider.IsExplicitGrokFreeOAuthProvider(gatewayprovider.ExecutionProtocolRecord(&providers[i]))}
	}
	blocked := gate.Blocked(candidates)
	if blocked == nil {
		return providers
	}
	filtered := make([]gatewayprovider.ExecutionProvider, 0, len(providers))
	for i, candidate := range candidates {
		if !candidate.Eligible || !blocked[candidate.ID] {
			filtered = append(filtered, providers[i])
		}
	}
	return filtered
}
