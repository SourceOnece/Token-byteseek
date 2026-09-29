package provider

import (
	"context"
	"fmt"
	"time"
)

// AntigravityUsageOptions 只提供供应商资格、查询与报文诊断，不持有缓存、规则或后台状态。
type AntigravityUsageOptions struct {
	CanFetch func(*Record) bool
	Fetch    func(context.Context, *Record) (*UsageInfo, error)
	Degrade  func(error, time.Time) *UsageInfo
	Enrich   func(*UsageInfo, *Record)
}

// antigravityUsageFlightResult 保留原提供商 flight key，等待者必须核对结果的来源身份。
type antigravityUsageFlightResult struct {
	Usage    *UsageInfo
	Identity string
}

// getAntigravityUsage 获取 Antigravity 提供商额度
func (s *OAuthUsageService) GetAntigravityUsage(ctx context.Context, provider *Record) (*UsageInfo, error) {
	if s.options.Antigravity.CanFetch == nil || !s.options.Antigravity.CanFetch(provider) {
		now := s.options.Now()
		return &UsageInfo{UpdatedAt: &now}, nil
	}

	identity := UsageCacheIdentity(provider)
	// 1. 检查缓存
	if cached, ok := s.cache.LoadAntigravity(provider.ID); ok {
		if cache, ok := cached.(*OAuthAntigravityUsageCache); ok && cache != nil && cache.Identity == identity {
			ttl := AntigravityCacheTTL(cache.UsageInfo)
			if s.options.Now().Sub(cache.Timestamp) < ttl {
				usage := CloneUsageInfo(cache.UsageInfo)
				if usage.FiveHour != nil && usage.FiveHour.ResetsAt != nil {
					usage.FiveHour.RemainingSeconds = int(usage.FiveHour.ResetsAt.Sub(s.options.Now()).Seconds())
				}
				return CloneUsageInfo(usage), nil
			}
		}
	}

	// 2. singleflight 防止并发击穿
	flightKey := fmt.Sprintf("ag-usage:%d", provider.ID)
	result, flightErr, _ := s.cache.DoAntigravity(flightKey, func() (any, error) {
		// 再次检查缓存（等待期间可能已被填充）
		if cached, ok := s.cache.LoadAntigravity(provider.ID); ok {
			if cache, ok := cached.(*OAuthAntigravityUsageCache); ok && cache != nil && cache.Identity == identity {
				ttl := AntigravityCacheTTL(cache.UsageInfo)
				if s.options.Now().Sub(cache.Timestamp) < ttl {
					usage := CloneUsageInfo(cache.UsageInfo)
					// 重新计算 RemainingSeconds，避免返回过时的剩余秒数
					RecalcAntigravityRemainingSeconds(usage, s.options.Now)
					return antigravityUsageFlightResult{CloneUsageInfo(usage), identity}, nil
				}
			}
		}

		// 使用独立 context，避免调用方 cancel 导致所有共享 flight 的请求失败
		fetchCtx, fetchCancel, beginErr := s.BeginDetached(ctx, 30*time.Second)
		if beginErr != nil {
			return nil, beginErr
		}
		defer fetchCancel()

		fetchedUsage, err := s.options.Antigravity.Fetch(fetchCtx, provider)
		if fetchCtx.Err() == context.Canceled {
			return nil, fetchCtx.Err()
		}
		if err != nil {
			degraded := s.options.Antigravity.Degrade(err, s.options.Now())
			s.options.Antigravity.Enrich(degraded, provider)
			s.cache.StoreAntigravity(provider.ID, &OAuthAntigravityUsageCache{
				Identity:  identity,
				UsageInfo: CloneUsageInfo(degraded),
				Timestamp: s.options.Now(),
			})
			return antigravityUsageFlightResult{CloneUsageInfo(degraded), identity}, nil
		}

		s.options.Antigravity.Enrich(fetchedUsage, provider)
		s.cache.StoreAntigravity(provider.ID, &OAuthAntigravityUsageCache{
			Identity:  identity,
			UsageInfo: CloneUsageInfo(fetchedUsage),
			Timestamp: s.options.Now(),
		})
		return antigravityUsageFlightResult{CloneUsageInfo(fetchedUsage), identity}, nil
	})

	if flightErr != nil {
		return nil, flightErr
	}
	completed, ok := result.(antigravityUsageFlightResult)
	if !ok || completed.Usage == nil {
		now := s.options.Now()
		return &UsageInfo{UpdatedAt: &now}, nil
	}
	if completed.Identity != identity {
		return nil, ErrUsageObservationChanged
	}
	return CloneUsageInfo(completed.Usage), nil
}
