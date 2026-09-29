package provider

import (
	"context"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
)

// AvailabilityProviders 只读取持久配置候选，不使用瞬时调度缓存或执行凭据入口。
type AvailabilityProviders interface {
	ListModelAvailabilityCandidates(context.Context, *int64, []string, bool) ([]provider.Record, error)
}

// NewModelAvailability 绑定提供商查询与分组映射读取端口，不维护独立缓存。
// @project-doc docs/architecture/provider_scheduling_and_cache.md#advanced_scheduler_selection
func NewModelAvailability(source AvailabilityProviders, groupPolicies *routing.PricingConfigService, compatible bool) *routing.ModelAvailability {
	result := &routing.ModelAvailability{
		MapModel: groupPolicies.ResolveRoutingModel,
	}
	if source == nil {
		return result
	}
	result.Read = func(ctx context.Context, group *int64, platforms []string, grouped bool) ([]routing.AvailabilityProvider, error) {
		if forced, ok := apikey.ForcePlatformFromContext(ctx); ok && strings.TrimSpace(forced) != "" {
			platforms = []string{forced}
		}
		values, err := source.ListModelAvailabilityCandidates(ctx, group, platforms, grouped)
		if err != nil {
			return nil, err
		}
		out := make([]routing.AvailabilityProvider, len(values))
		for i := range values {
			record := &values[i]
			out[i] = routing.AvailabilityProvider{
				Platform: record.Platform,
				Supports: func(ctx context.Context, model string) bool {
					policy := ModelPolicy{Record: record}
					if !policy.AllowsProtocol(ctx) {
						return false
					}
					if compatible {
						return policy.SupportsCompatibleRouting(ctx, model)
					}
					return policy.Supports(ctx, model)
				},
			}
		}
		return out, nil
	}
	return result
}

// SupportsCompatibleRouting 使用提供商平台的模型能力规则，透传提供商也受模型范围约束。
func (p ModelPolicy) SupportsCompatibleRouting(ctx context.Context, model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return true
	}
	if p.Record == nil {
		return false
	}
	return p.Supports(ctx, model)
}
