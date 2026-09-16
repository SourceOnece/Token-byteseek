package service

// 模型 429 触发有界后台探测；复用已有按 API Key 分组刷新、并发控制和退避，不阻塞模型请求。
// 队列合并同账号请求，单协调器随服务启动/停止；只有新鲜且完整的耗尽快照才回调。

import (
	"context"
	"time"
)

const (
	ollamaCloudUsageProbeTimeout = 45 * time.Second

	ollamaCloudUsageProbeMaxQueue = 256

	ollamaCloudUsageProbeGroupRetention = 2 * ollamaCloudUsageManualRefreshInterval
)

type OllamaCloudUsageRateLimitProbeCallback func(accountID int64, resetAt time.Time)

type ollamaCloudUsageProbeRequest struct {
	accountID   int64
	onExhausted OllamaCloudUsageRateLimitProbeCallback
}

type ollamaCloudUsageProbeGroupEntry struct {
	attemptAt time.Time
	snapshot  *OllamaCloudUsageSnapshot
}

func (s *OllamaCloudUsageService) ScheduleOllamaCloudUsageRateLimitProbe(
	accountID int64,
	onExhausted OllamaCloudUsageRateLimitProbeCallback,
) bool {
	if s == nil || onExhausted == nil || accountID <= 0 {
		return false
	}
	s.mu.Lock()
	running := s.started && !s.stopped
	s.mu.Unlock()
	if !running {
		return false
	}

	s.probeMu.Lock()
	for i := range s.probeQueue {
		if s.probeQueue[i].accountID == accountID {
			s.probeQueue[i].onExhausted = onExhausted
			s.probeMu.Unlock()
			s.wakeProbeLoop()
			return true
		}
	}
	if len(s.probeQueue) >= ollamaCloudUsageProbeMaxQueue {
		s.probeMu.Unlock()
		return false
	}
	s.probeQueue = append(s.probeQueue, ollamaCloudUsageProbeRequest{
		accountID:   accountID,
		onExhausted: onExhausted,
	})
	s.probeMu.Unlock()

	s.wakeProbeLoop()
	return true
}

func (s *OllamaCloudUsageService) wakeProbeLoop() {
	if s == nil {
		return
	}
	select {
	case s.probeWake <- struct{}{}:
	default:
	}
}

func (s *OllamaCloudUsageService) probeLoop() {
	defer s.wg.Done()
	for {
		s.probeMu.Lock()
		if len(s.probeQueue) == 0 {
			s.probeMu.Unlock()
			select {
			case <-s.probeWake:
			case <-s.parentCtx.Done():
				return
			}
			continue
		}
		batch := s.probeQueue
		s.probeQueue = nil
		s.probeMu.Unlock()

		for _, req := range batch {
			if s.parentCtx.Err() != nil {
				return
			}
			ctx, cancel := context.WithTimeout(s.parentCtx, ollamaCloudUsageProbeTimeout)
			s.runOllamaCloudUsageProbe(ctx, req.accountID, req.onExhausted)
			cancel()
		}
	}
}

func (s *OllamaCloudUsageService) probeGroupResult(key string) (ollamaCloudUsageProbeGroupEntry, bool) {
	if s == nil {
		return ollamaCloudUsageProbeGroupEntry{}, false
	}
	s.probeMu.Lock()
	defer s.probeMu.Unlock()
	entry, ok := s.probeGroups[key]
	return entry, ok
}

func (s *OllamaCloudUsageService) storeProbeGroupResult(key string, entry ollamaCloudUsageProbeGroupEntry) {
	if s == nil {
		return
	}
	s.probeMu.Lock()
	defer s.probeMu.Unlock()
	if s.probeGroups == nil {
		s.probeGroups = make(map[string]ollamaCloudUsageProbeGroupEntry)
	}
	pruneAt := entry.attemptAt.Add(-ollamaCloudUsageProbeGroupRetention)
	for groupKey, existing := range s.probeGroups {
		if existing.attemptAt.Before(pruneAt) {
			delete(s.probeGroups, groupKey)
		}
	}
	s.probeGroups[key] = entry
}

func (s *OllamaCloudUsageService) runOllamaCloudUsageProbe(
	ctx context.Context,
	accountID int64,
	onExhausted OllamaCloudUsageRateLimitProbeCallback,
) {
	if s == nil || s.accountRepo == nil || onExhausted == nil || ctx.Err() != nil {
		return
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil || account == nil {
		return
	}
	if !IsOllamaCloudUsageAccount(account) {
		return
	}
	if err := s.ResolveAccounts(ctx, []*Account{account}); err != nil {
		return
	}
	if !ollamaCloudUsageConfigured(account) {
		return
	}
	key, valid := ollamaCloudUsageGroupFingerprint(account)
	if !valid {
		return
	}

	window := ollamaCloudUsageManualRefreshInterval
	accountSnapshot := decodeOllamaCloudUsageSnapshot(account.Extra)
	cached, hasCached := s.probeGroupResult(key)

	now := s.currentTime()
	if newest := ollamaCloudUsageProbeNewestSuccess(accountSnapshot, cached, hasCached, now, window); newest != nil {
		maybeOllamaCloudUsageProbeExhaustion(ctx, accountID, newest, s.currentTime(), window, onExhausted)
		return
	}

	if horizon := ollamaCloudUsageProbeBackoffHorizon(accountSnapshot, cached, hasCached); !horizon.IsZero() && now.Before(horizon) {
		return
	}
	if hasCached && now.Before(cached.attemptAt.Add(window)) {
		return
	}

	settings, settingsErr := s.GetSettings(ctx)
	if settingsErr != nil {
		return
	}
	fetched, refreshErr := s.refreshAccount(ctx, accountID, settings, false)
	if refreshErr == nil && fetched == nil {
		return
	}
	doneNow := s.currentTime()
	s.storeProbeGroupResult(key, ollamaCloudUsageProbeGroupEntry{attemptAt: doneNow, snapshot: fetched})
	if refreshErr != nil {
		return
	}
	maybeOllamaCloudUsageProbeExhaustion(ctx, accountID, fetched, doneNow, window, onExhausted)
}

func ollamaCloudUsageProbeObservedAt(snapshot *OllamaCloudUsageSnapshot) (time.Time, bool) {
	if snapshot == nil {
		return time.Time{}, false
	}
	if snapshot.Status == OllamaCloudUsageStatusOK && snapshot.FetchedAt != nil && !snapshot.FetchedAt.IsZero() {
		return snapshot.FetchedAt.UTC(), true
	}
	if !snapshot.LastAttemptAt.IsZero() {
		return snapshot.LastAttemptAt.UTC(), true
	}
	return time.Time{}, false
}

func ollamaCloudUsageProbeNewestSuccess(
	accountSnapshot *OllamaCloudUsageSnapshot,
	cached ollamaCloudUsageProbeGroupEntry,
	hasCached bool,
	now time.Time,
	window time.Duration,
) *OllamaCloudUsageSnapshot {
	var newest *OllamaCloudUsageSnapshot
	var newestAt time.Time
	consider := func(snapshot *OllamaCloudUsageSnapshot) {
		if snapshot == nil {
			return
		}
		at, ok := ollamaCloudUsageProbeObservedAt(snapshot)
		if !ok {
			return
		}
		if newest == nil || at.After(newestAt) {
			newest = snapshot
			newestAt = at
		}
	}
	consider(accountSnapshot)
	if hasCached {
		if cached.snapshot != nil {
			consider(cached.snapshot)
		} else if !cached.attemptAt.IsZero() && (newest == nil || cached.attemptAt.After(newestAt)) {
			newest = nil
			newestAt = cached.attemptAt
		}
	}
	if newest == nil || newest.Status != OllamaCloudUsageStatusOK {
		return nil
	}
	if newest.FetchedAt == nil || newest.FetchedAt.IsZero() || newest.FetchedAt.Before(now.Add(-window)) {
		return nil
	}
	return newest
}

func ollamaCloudUsageProbeBackoffHorizon(
	accountSnapshot *OllamaCloudUsageSnapshot,
	cached ollamaCloudUsageProbeGroupEntry,
	hasCached bool,
) time.Time {
	var horizon time.Time
	consider := func(snapshot *OllamaCloudUsageSnapshot) {
		if snapshot == nil {
			return
		}
		if snapshot.Status != OllamaCloudUsageStatusFailed && snapshot.Status != OllamaCloudUsageStatusUnauthorized {
			return
		}
		if snapshot.NextRefreshAt.IsZero() {
			return
		}
		if horizon.IsZero() || snapshot.NextRefreshAt.After(horizon) {
			horizon = snapshot.NextRefreshAt.UTC()
		}
	}
	consider(accountSnapshot)
	if hasCached {
		consider(cached.snapshot)
	}
	return horizon
}

func maybeOllamaCloudUsageProbeExhaustion(
	ctx context.Context,
	accountID int64,
	snapshot *OllamaCloudUsageSnapshot,
	now time.Time,
	window time.Duration,
	onExhausted OllamaCloudUsageRateLimitProbeCallback,
) {
	if snapshot == nil || onExhausted == nil {
		return
	}
	resetAt, exhausted := ollamaCloudUsageExhaustionResetAt(snapshot, now, now.Add(-window))
	if !exhausted {
		return
	}
	if ctx.Err() != nil {
		return
	}
	onExhausted(accountID, resetAt)
}
