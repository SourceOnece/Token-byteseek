package httpapi

import (
	"context"
	"sync"

	"github.com/TokenFlux/TokenRouter/internal/scheduler"
)

// 此文件只有既有并发存储端口的测试计数，不实现等待、计费或租约规则。
type helperConcurrencyCacheStub struct {
	mu sync.Mutex

	providerSeq []bool
	userSeq     []bool

	providerAcquireCalls int
	userAcquireCalls     int
	providerReleaseCalls int
	userReleaseCalls     int
	waitAllowed          bool
	waitIncrementCalls   int
	waitDecrementCalls   int
	waitMaxWait          int
	waitIncrementHook    func()
	apiKeyTrackCalls     int
	apiKeyReleaseCalls   int
	apiKeyTrackIDs       []int64
}

func (s *helperConcurrencyCacheStub) AcquireProviderSlot(ctx context.Context, providerID int64, maxConcurrency int, requestID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providerAcquireCalls++
	if len(s.providerSeq) == 0 {
		return false, nil
	}
	v := s.providerSeq[0]
	s.providerSeq = s.providerSeq[1:]
	return v, nil
}

func (s *helperConcurrencyCacheStub) ReleaseProviderSlot(ctx context.Context, providerID int64, requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providerReleaseCalls++
	return nil
}

func (s *helperConcurrencyCacheStub) GetProviderConcurrency(ctx context.Context, providerID int64) (int, error) {
	return 0, nil
}

func (s *helperConcurrencyCacheStub) GetProviderConcurrencyBatch(ctx context.Context, providerIDs []int64) (map[int64]int, error) {
	out := make(map[int64]int, len(providerIDs))
	for _, providerID := range providerIDs {
		out[providerID] = 0
	}
	return out, nil
}

func (s *helperConcurrencyCacheStub) IncrementProviderWaitCount(ctx context.Context, providerID int64, maxWait int) (bool, error) {
	return true, nil
}

func (s *helperConcurrencyCacheStub) DecrementProviderWaitCount(ctx context.Context, providerID int64) error {
	return nil
}

func (s *helperConcurrencyCacheStub) GetProviderWaitingCount(ctx context.Context, providerID int64) (int, error) {
	return 0, nil
}

func (s *helperConcurrencyCacheStub) AcquireUserSlot(ctx context.Context, userID int64, maxConcurrency int, requestID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userAcquireCalls++
	if len(s.userSeq) == 0 {
		return false, nil
	}
	v := s.userSeq[0]
	s.userSeq = s.userSeq[1:]
	return v, nil
}

func (s *helperConcurrencyCacheStub) ReleaseUserSlot(ctx context.Context, userID int64, requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userReleaseCalls++
	return nil
}

func (s *helperConcurrencyCacheStub) GetUserConcurrency(ctx context.Context, userID int64) (int, error) {
	return 0, nil
}

func (s *helperConcurrencyCacheStub) TrackAPIKeySlot(ctx context.Context, apiKeyID int64, requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apiKeyTrackCalls++
	s.apiKeyTrackIDs = append(s.apiKeyTrackIDs, apiKeyID)
	return nil
}

func (s *helperConcurrencyCacheStub) ReleaseAPIKeySlot(ctx context.Context, apiKeyID int64, requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apiKeyReleaseCalls++
	return nil
}

func (s *helperConcurrencyCacheStub) GetAPIKeyConcurrencyBatch(ctx context.Context, apiKeyIDs []int64) (map[int64]int, error) {
	out := make(map[int64]int, len(apiKeyIDs))
	for _, apiKeyID := range apiKeyIDs {
		out[apiKeyID] = 0
	}
	return out, nil
}

func (s *helperConcurrencyCacheStub) IncrementWaitCount(ctx context.Context, userID int64, maxWait int) (bool, error) {
	s.mu.Lock()
	s.waitIncrementCalls++
	s.waitMaxWait = maxWait
	waitAllowed := s.waitAllowed
	hook := s.waitIncrementHook
	s.mu.Unlock()

	if hook != nil {
		hook()
	}
	if !waitAllowed {
		return false, nil
	}
	return true, nil
}

func (s *helperConcurrencyCacheStub) DecrementWaitCount(ctx context.Context, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.waitDecrementCalls++
	return nil
}

func (s *helperConcurrencyCacheStub) GetProvidersLoadBatch(ctx context.Context, providers []scheduler.ProviderWithConcurrency) (map[int64]*scheduler.ProviderLoadInfo, error) {
	out := make(map[int64]*scheduler.ProviderLoadInfo, len(providers))
	for _, acc := range providers {
		out[acc.ID] = &scheduler.ProviderLoadInfo{ProviderID: acc.ID}
	}
	return out, nil
}

func (s *helperConcurrencyCacheStub) GetUsersLoadBatch(ctx context.Context, users []scheduler.UserWithConcurrency) (map[int64]*scheduler.UserLoadInfo, error) {
	out := make(map[int64]*scheduler.UserLoadInfo, len(users))
	for _, user := range users {
		out[user.ID] = &scheduler.UserLoadInfo{UserID: user.ID}
	}
	return out, nil
}

func (s *helperConcurrencyCacheStub) CleanupExpiredProviderSlots(ctx context.Context, providerID int64) error {
	return nil
}

func (s *helperConcurrencyCacheStub) CleanupExpiredProviderSlotKeys(ctx context.Context) error {
	return nil
}

func (s *helperConcurrencyCacheStub) CleanupStaleProcessSlots(ctx context.Context, activeRequestPrefix string) error {
	return nil
}
