package billing

import (
	"context"
	"maps"
	"sync/atomic"
	"time"
)

// CostWindowInput 是调度消费者的标准费用窗口投影，不包含凭据或旧提供商实体。
type CostWindowInput struct {
	ID             int64
	Enabled        bool
	Limit, Reserve float64
	Start, End     *time.Time
}
type (
	WindowCostStats  struct{ StandardCost float64 }
	WindowCostSource interface {
		GetWindow(context.Context, int64, time.Time) (*WindowCostStats, error)
	}
)

type WindowCostBatchSource interface {
	GetWindows(context.Context, []int64, time.Time) (map[int64]*WindowCostStats, error)
}
type WindowCostMetrics struct{ Hit, Miss, BatchSQL, Fallback, Errors atomic.Int64 }

var sharedWindowCostMetrics WindowCostMetrics

func SharedWindowCostMetrics() *WindowCostMetrics { return &sharedWindowCostMetrics }

type WindowCostGuardOptions struct {
	Now   func() time.Time
	Log   func(string, ...any)
	Debug func(string, ...any)
	Stats *WindowCostMetrics
}

// WindowCostGuard 复用既有 Redis 缓存，按需查询窗口标准费用；不会持有第二份缓存或写入资金。
type WindowCostGuard struct {
	cache  WindowCostCache
	source WindowCostSource
	now    func() time.Time
	log    func(string, ...any)
	debug  func(string, ...any)
	stats  *WindowCostMetrics
}

func NewWindowCostGuard(cache WindowCostCache, source WindowCostSource, options WindowCostGuardOptions) *WindowCostGuard {
	return &WindowCostGuard{cache: cache, source: source, now: options.Now, log: options.Log, debug: options.Debug, stats: options.Stats}
}

type windowCostPrefetchKey struct{}

func WithPrefetchedWindowCosts(ctx context.Context, costs map[int64]float64) context.Context {
	return context.WithValue(ctx, windowCostPrefetchKey{}, maps.Clone(costs))
}

func PrefetchedWindowCost(ctx context.Context, id int64) (float64, bool) {
	if ctx == nil || id <= 0 {
		return 0, false
	}
	m, ok := ctx.Value(windowCostPrefetchKey{}).(map[int64]float64)
	if !ok || len(m) == 0 {
		return 0, false
	}
	v, found := m[id]
	return v, found
}

func (s *WindowCostGuard) Prefetch(ctx context.Context, providers []CostWindowInput) context.Context {
	if ctx == nil || len(providers) == 0 || s.cache == nil || s.source == nil {
		return ctx
	}

	providerByID := make(map[int64]*CostWindowInput)
	providerIDs := make([]int64, 0, len(providers))
	for i := range providers {
		provider := &providers[i]
		if provider == nil || !provider.Enabled {
			continue
		}
		if provider.Limit <= 0 {
			continue
		}
		providerByID[provider.ID] = provider
		providerIDs = append(providerIDs, provider.ID)
	}
	if len(providerIDs) == 0 {
		return ctx
	}

	costs := make(map[int64]float64, len(providerIDs))
	cacheValues, err := s.cache.GetWindowCostBatch(ctx, providerIDs)
	if err == nil {
		for providerID, cost := range cacheValues {
			costs[providerID] = cost
		}
		s.stats.Hit.Add(int64(len(cacheValues)))
	} else {
		s.stats.Errors.Add(1)
		s.log("window_cost batch cache read failed: %v", err)
	}
	cacheMissCount := len(providerIDs) - len(costs)
	if cacheMissCount < 0 {
		cacheMissCount = 0
	}
	s.stats.Miss.Add(int64(cacheMissCount))

	missingByStart := make(map[int64][]int64)
	startTimes := make(map[int64]time.Time)
	for _, providerID := range providerIDs {
		if _, ok := costs[providerID]; ok {
			continue
		}
		provider := providerByID[providerID]
		if provider == nil {
			continue
		}
		startTime := CurrentCostWindowStart(provider.Start, provider.End, s.now())
		startKey := startTime.Unix()
		missingByStart[startKey] = append(missingByStart[startKey], providerID)
		startTimes[startKey] = startTime
	}
	if len(missingByStart) == 0 {
		return WithPrefetchedWindowCosts(ctx, costs)
	}

	batchReader, hasBatch := s.source.(WindowCostBatchSource)
	for startKey, ids := range missingByStart {
		startTime := startTimes[startKey]

		if hasBatch {
			s.stats.BatchSQL.Add(1)
			queryStart := s.now()
			statsByProvider, err := batchReader.GetWindows(ctx, ids, startTime)
			if err == nil {
				s.debug("window_cost_batch_query_ok",
					"providers", len(ids),
					"window_start", startTime.Format(time.RFC3339),
					"duration_ms", s.now().Sub(queryStart).Milliseconds())
				for _, providerID := range ids {
					stats := statsByProvider[providerID]
					cost := 0.0
					if stats != nil {
						cost = stats.StandardCost
					}
					costs[providerID] = cost
					_ = s.cache.SetWindowCost(ctx, providerID, cost)
				}
				continue
			}
			s.stats.Errors.Add(1)
			s.log("window_cost batch db query failed: start=%s err=%v", startTime.Format(time.RFC3339), err)
		}

		// 回退路径：缺少批量仓储能力或批量查询失败时，按提供商单查（失败开放）。
		s.stats.Fallback.Add(int64(len(ids)))
		for _, providerID := range ids {
			stats, err := s.source.GetWindow(ctx, providerID, startTime)
			if err != nil {
				s.stats.Errors.Add(1)
				continue
			}
			cost := stats.StandardCost
			costs[providerID] = cost
			_ = s.cache.SetWindowCost(ctx, providerID, cost)
		}
	}

	return WithPrefetchedWindowCosts(ctx, costs)
}

func (s *WindowCostGuard) Allow(ctx context.Context, provider CostWindowInput, isSticky bool) bool {
	// 只检查 Anthropic OAuth/SetupToken 提供商
	if !provider.Enabled {
		return true
	}

	limit := provider.Limit
	if limit <= 0 {
		return true // 未启用窗口费用限制
	}

	// 尝试从缓存获取窗口费用
	var currentCost float64
	if cost, ok := PrefetchedWindowCost(ctx, provider.ID); ok {
		currentCost = cost
		goto checkSchedulability
	}
	if s.cache != nil {
		if cost, hit, err := s.cache.GetWindowCost(ctx, provider.ID); err == nil && hit {
			currentCost = cost
			goto checkSchedulability
		}
	}

	// 缓存未命中，从数据库查询
	{
		// 使用统一的窗口开始时间计算逻辑（考虑窗口过期情况）
		startTime := CurrentCostWindowStart(provider.Start, provider.End, s.now())

		stats, err := s.source.GetWindow(ctx, provider.ID, startTime)
		if err != nil {
			// 失败开放：查询失败时允许调度
			return true
		}

		// 使用标准费用（不含提供商倍率）
		currentCost = stats.StandardCost

		// 设置缓存（忽略错误）
		if s.cache != nil {
			_ = s.cache.SetWindowCost(ctx, provider.ID, currentCost)
		}
	}

checkSchedulability:
	schedulability := CheckWindowCost(currentCost, provider.Limit, provider.Reserve)

	switch schedulability {
	case WindowCostSchedulable:
		return true
	case WindowCostStickyOnly:
		return isSticky
	case WindowCostNotSchedulable:
		return false
	}
	return true
}
