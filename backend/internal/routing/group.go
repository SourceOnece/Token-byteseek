package routing

import (
	"fmt"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/routing/accessview"
	"github.com/TokenFlux/TokenRouter/internal/scheduler/policy"
)

type GroupModelsListConfig = accessview.GroupModelsListConfig

type GroupAvailabilityProbeConfig = accessview.GroupAvailabilityProbeConfig

type GroupAdvancedSchedulerOverrides = policy.GroupAdvancedSchedulerOverrides

type GroupSchedulerType = accessview.GroupSchedulerType

const (
	// GroupSchedulerTypeBasic 保持当前默认调度路径。
	GroupSchedulerTypeBasic GroupSchedulerType = "basic"
	// GroupSchedulerTypeAdvanced 使用通用高级调度器。
	GroupSchedulerTypeAdvanced GroupSchedulerType = "advanced"
)

// NormalizeGroupSchedulerType 归一化并校验调度器类型。
func NormalizeGroupSchedulerType(value string) (GroupSchedulerType, error) {
	normalized := GroupSchedulerType(strings.ToLower(strings.TrimSpace(value)))
	switch normalized {
	case "", GroupSchedulerTypeBasic:
		return GroupSchedulerTypeBasic, nil
	case GroupSchedulerTypeAdvanced:
		return GroupSchedulerTypeAdvanced, nil
	default:
		return "", fmt.Errorf("scheduler_type must be basic or advanced")
	}
}

// Group 在共享值契约上拥有分组规则，提供商只读取 accessview 的值投影。
type GroupRoutingPolicy = accessview.GroupRoutingPolicy

type Group accessview.GroupConfig

// UsesAdvancedScheduler 返回分组是否启用通用高级调度器。
func (g *Group) UsesAdvancedScheduler() bool {
	return g != nil && g.SchedulerType == GroupSchedulerTypeAdvanced
}

func (g *Group) IsActive() bool {
	return g.Status == StatusActive
}

// IsGroupContextValid reports whether a group from context has the fields required for routing decisions.
func IsGroupContextValid(group *Group) bool {
	if group == nil {
		return false
	}
	if group.ID <= 0 {
		return false
	}
	if !group.Hydrated {
		return false
	}
	if group.Status == "" {
		return false
	}
	return true
}

// GetRoutingProviderIDs 根据请求模型获取路由提供商 ID 列表
// 返回匹配的优先提供商 ID 列表，如果没有匹配规则则返回 nil
func (g *Group) GetRoutingProviderIDs(requestedModel string) []int64 {
	if !g.ModelRoutingEnabled || len(g.ModelRouting) == 0 || requestedModel == "" {
		return nil
	}

	// 1. 精确匹配优先
	if providerIDs, ok := g.ModelRouting[requestedModel]; ok && len(providerIDs) > 0 {
		return providerIDs
	}

	// 2. 通配符匹配（前缀匹配）
	for pattern, providerIDs := range g.ModelRouting {
		if MatchModelPattern(pattern, requestedModel) && len(providerIDs) > 0 {
			return providerIDs
		}
	}

	return nil
}

// MatchModelPattern 检查模型是否匹配模式
// 支持 * 通配符，如 "claude-opus-*" 匹配 "claude-opus-4-20250514"
func MatchModelPattern(pattern, model string) bool {
	if pattern == model {
		return true
	}

	// 处理 * 通配符（仅支持末尾通配符）
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(model, prefix)
	}

	return false
}
