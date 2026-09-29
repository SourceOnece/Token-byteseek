package provider

import (
	"context"
	"sync"
	"time"
)

// OAuthRefreshCandidatePager 沿用原有界分页与原始 ID 游标，不退回无界扫描。
type OAuthRefreshCandidatePager interface {
	ListOAuthRefreshCandidatePage(context.Context, OAuthRefreshPageOptions) (*OAuthRefreshCandidatePage, error)
}

// RefreshScanOptions 为单轮配置投影与观察端口，不包含具体配置或供应商客户端。
type RefreshScanOptions struct {
	Timeout                  time.Duration
	PageSize                 int
	Platforms                []string
	Debug, Info, Warn, Error func(string, ...any)
}

// RefreshCandidateScan 持有唯一恢复游标；周期由 app 注入的提供商刷新循环驱动。
type RefreshCandidateScan struct {
	mu      sync.Mutex
	afterID int64
}

func (s *RefreshCandidateScan) Position() int64      { s.mu.Lock(); defer s.mu.Unlock(); return s.afterID }
func (s *RefreshCandidateScan) SetPosition(id int64) { s.mu.Lock(); s.afterID = id; s.mu.Unlock() }

type RefreshPageStats struct {
	Total        int
	OAuth        int
	NeedsRefresh int
	Refreshed    int
	Skipped      int
	Failed       int
}

// Run 执行一次有界且可从游标恢复的刷新周期。
func (s *RefreshCandidateScan) Run(parent context.Context, pager OAuthRefreshCandidatePager, options RefreshScanOptions, process func(context.Context, []Record) RefreshPageStats) {
	noop := func(string, ...any) {}
	if options.Debug == nil {
		options.Debug = noop
	}
	if options.Info == nil {
		options.Info = noop
	}
	if options.Warn == nil {
		options.Warn = noop
	}
	if options.Error == nil {
		options.Error = noop
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, options.Timeout)
	defer cancel()

	if pager == nil {
		options.Error("token_refresh.candidate_pager_missing")
		return
	}
	platforms := options.Platforms
	if len(platforms) == 0 {
		options.Error("token_refresh.provider_registry_empty")
		return
	}

	pageSize := options.PageSize
	stats := RefreshPageStats{}
	afterID := s.Position()
	for {
		if ctx.Err() != nil {
			options.Warn("token_refresh.cycle_stopped", "error", ctx.Err(), "resume_after_id", afterID)
			break
		}
		page, err := pager.ListOAuthRefreshCandidatePage(ctx, OAuthRefreshPageOptions{
			Platforms:            platforms,
			AfterID:              afterID,
			Limit:                pageSize,
			ActiveOnly:           true,
			IncludeSetupToken:    true,
			RequireRefreshToken:  true,
			ExcludeRetryCooldown: true,
		})
		if err != nil {
			options.Error("token_refresh.list_providers_failed", "error", err, "after_id", afterID)
			break
		}
		if page == nil {
			options.Error("token_refresh.nil_candidate_page", "after_id", afterID)
			break
		}
		providers := page.Providers
		if !page.HasMore && page.NextAfterID == 0 && len(providers) == 0 {
			s.SetPosition(0)
			break
		}
		if page.NextAfterID <= afterID {
			options.Error("token_refresh.invalid_candidate_page_metadata", "after_id", afterID)
			break
		}
		if !IsStrictlyIncreasingProviderPage(providers, afterID) {
			options.Error("token_refresh.invalid_candidate_page", "after_id", afterID, "count", len(providers))
			break
		}

		pageStats := process(ctx, providers)
		stats.Total += pageStats.Total
		stats.OAuth += pageStats.OAuth
		stats.NeedsRefresh += pageStats.NeedsRefresh
		stats.Refreshed += pageStats.Refreshed
		stats.Skipped += pageStats.Skipped
		stats.Failed += pageStats.Failed

		// 页面未完整处理时不得推进游标；OAuthRefreshAPI 会重读数据库并复查过期时间，因此重读页面是安全的。
		if ctx.Err() != nil {
			break
		}
		afterID = page.NextAfterID
		s.SetPosition(afterID)
		if !page.HasMore {
			s.SetPosition(0)
			break
		}
	}

	if stats.NeedsRefresh == 0 && stats.Failed == 0 {
		options.Debug("token_refresh.cycle_completed",
			"total", stats.Total, "oauth", stats.OAuth,
			"needs_refresh", stats.NeedsRefresh, "refreshed", stats.Refreshed,
			"skipped", stats.Skipped, "failed", stats.Failed)
	} else {
		options.Info("token_refresh.cycle_completed",
			"total", stats.Total, "oauth", stats.OAuth,
			"needs_refresh", stats.NeedsRefresh, "refreshed", stats.Refreshed,
			"skipped", stats.Skipped, "failed", stats.Failed)
	}
}

func IsStrictlyIncreasingProviderPage(providers []Record, afterID int64) bool {
	previous := afterID
	for i := range providers {
		if providers[i].ID <= previous {
			return false
		}
		previous = providers[i].ID
	}
	return true
}
