package dto

import (
	"github.com/TokenFlux/TokenRouter/internal/protocol"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/scheduler/policy"
)

func GroupFromRoutingBase(g *routing.Group) Group {
	return Group{
		Models: append([]string{}, g.Models...), ModelProtocols: g.ModelProtocols,
		ID:                              g.ID,
		Name:                            g.Name,
		Description:                     g.Description,
		DisplayBrand:                    g.DisplayBrand,
		RateMultiplier:                  g.RateMultiplier,
		IsExclusive:                     g.IsExclusive,
		Status:                          g.Status,
		SessionIsolationEnabled:         g.SessionIsolationEnabled,
		AllowImageGeneration:            g.AllowImageGeneration,
		AllowBatchImageGeneration:       g.AllowBatchImageGeneration,
		ClaudeCodeOnly:                  g.ClaudeCodeOnly,
		FallbackGroupID:                 g.FallbackGroupID,
		FallbackGroupIDOnInvalidRequest: g.FallbackGroupIDOnInvalidRequest,
		UnavailableFallbackGroupID:      g.UnavailableFallbackGroupID,
		AllowedProtocols:                g.EffectiveAllowedProtocols(),
		ProtocolFallbacks:               g.ProtocolFallbacks,
		ResponsesImagePolicy:            g.ResponsesImagePolicy,
		AllowMessagesDispatch:           g.AllowsClientProtocol(protocol.ProtocolAnthropicMessages),
		AllowLive:                       g.AllowLive,
		RequireOAuthOnly:                g.RequireOAuthOnly,
		RequirePrivacySet:               g.RequirePrivacySet,
		RPMLimit:                        g.RPMLimit,
		MaxReasoningEffort:              g.MaxReasoningEffort,
		MaxReasoningEffortOverLimit:     g.MaxReasoningEffortOverLimit,
		ReasoningEffortMappings:         g.ReasoningEffortMappings,
		CreatedAt:                       g.CreatedAt,
		UpdatedAt:                       g.UpdatedAt,
	}
}

// GroupFromServiceAdmin converts a service Group to DTO for admin users.
// It includes internal fields like model_routing and provider_count.
func AdminGroupFromRouting[A any](g *routing.Group) *AdminGroup[A] {
	if g == nil {
		return nil
	}
	out := &AdminGroup[A]{
		Group:                      GroupFromRoutingBase(g),
		ForceOpenAIFast:            g.ForceOpenAIFast,
		OpenAIFastPolicy:           g.EffectiveOpenAIFastPolicy(),
		SchedulerType:              string(g.SchedulerType),
		AdvancedSchedulerOverrides: policy.CloneGroupAdvancedSchedulerOverrides(g.AdvancedSchedulerOverrides),
		RoutingPolicy:              g.RoutingPolicy.Clone(),
		ModelRouting:               g.ModelRouting,
		ModelRoutingEnabled:        g.ModelRoutingEnabled,
		MCPXMLInject:               g.MCPXMLInject,
		DefaultMappedModel:         g.DefaultMappedModel,
		ModelsListConfig:           g.ModelsListConfig,
		ModelAllowlist:             g.ModelAllowlist,
		AvailabilityProbeConfig:    g.AvailabilityProbeConfig,
		SupportedModelScopes:       g.SupportedModelScopes,
		ProviderCount:              g.ProviderCount,
		ActiveProviderCount:        g.ActiveProviderCount,
		RateLimitedProviderCount:   g.RateLimitedProviderCount,
		SortOrder:                  g.SortOrder,
	}

	return out
}

// GroupFromRouting 不输出内部字段，保留用户与管理员 DTO 的字段边界。
func GroupFromRouting(g *routing.Group) *Group {
	if g == nil {
		return nil
	}
	out := GroupFromRoutingBase(g)
	return &out
}

// GroupCapacityFromSummary 仅投影既有容量字段，不改变 nil 与零值表示。
func GroupCapacityFromSummary(v *routing.GroupCapacitySummary) *GroupCapacity {
	if v == nil {
		return nil
	}
	return &GroupCapacity{ConcurrencyUsed: v.ConcurrencyUsed, ConcurrencyMax: v.ConcurrencyMax, SessionsUsed: v.SessionsUsed, SessionsMax: v.SessionsMax, RPMUsed: v.RPMUsed, RPMMax: v.RPMMax}
}
