package provider

import (
	"context"
	"errors"
	"time"

	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// QoderRequestRefresh 复用唯一提供商刷新协调器及会话缓存，拥有请求失败后的回读和失效顺序。
// 它不启动后台任务，也不创建额外锁或缓存。
type QoderRequestRefresh struct {
	Store        provider.RefreshRepository
	Tokens       *QoderTokenProvider
	Coordinator  *provider.OAuthRefreshAPI
	NewRefresher func() *QoderTokenRefresher
	Transport    QoderTransport
	Profiles     *egressprovider.TLSProfiles
}

func (s *QoderRequestRefresh) RefreshProviderSession(ctx context.Context, value *provider.Record) (*provider.Record, error) {
	if s == nil {
		return nil, errors.New("qoder gateway service is not configured")
	}
	if value == nil {
		return nil, errors.New("provider is nil")
	}
	if s.Store == nil {
		return nil, errors.New("qoder provider repository is not configured")
	}
	refresherFactory := s.NewRefresher
	if refresherFactory == nil {
		refresherFactory = func() *QoderTokenRefresher {
			return NewQoderTokenRefresher(QoderRefreshOptions{Transport: s.Transport, Profiles: s.Profiles})
		}
	}
	refresher := refresherFactory()
	if refresher == nil {
		return nil, errors.New("qoder token refresher is nil")
	}
	refreshAPI := s.Coordinator
	if refreshAPI == nil {
		return nil, errors.New("qoder refresh API is not configured")
	}
	failedCredentialsHash := provider.QoderRefreshCredentialsHash(value.Credentials)
	executor := qoderFailedRefreshExecutor{
		QoderTokenRefresher: refresher,
		failedCredentials:   failedCredentialsHash,
	}
	result, err := refreshAPI.RefreshIfNeeded(ctx, value, executor, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	// 如果另一个 worker 正在刷新（LockHeld=true），等待 DB 中出现已轮换凭证。
	// 不能只 sleep 后返回当前提供商：锁持有者可能尚未写回新 token，
	// handler 随后会用同一份 stale credentials 立即重试并再次 401。
	if result != nil && result.LockHeld {
		return s.waitForQoderLockedRefresh(ctx, value, failedCredentialsHash)
	}

	if result != nil && result.Provider != nil {
		if s.Tokens != nil {
			s.Tokens.InvalidateProvider(result.Provider)
		}
		return result.Provider, nil
	}
	if s.Store != nil {
		if fresh, err := s.Store.GetByID(ctx, value.ID); err == nil && fresh != nil {
			if s.Tokens != nil {
				s.Tokens.InvalidateProvider(fresh)
			}
			return fresh, nil
		}
	}
	if s.Tokens != nil && (result == nil || result.Refreshed) {
		s.Tokens.Invalidate(value.ID)
	}
	return value, nil
}

func (s *QoderRequestRefresh) waitForQoderLockedRefresh(ctx context.Context, value *provider.Record, failedCredentialsHash string) (*provider.Record, error) {
	if s == nil || s.Store == nil || value == nil {
		return nil, provider.ErrQoderRefreshInProgress
	}
	var result *provider.Record
	err := provider.WaitForQoderRefresh(ctx, func(readCtx context.Context) (bool, error) {
		fresh, err := s.Store.GetByID(readCtx, value.ID)
		if err != nil {
			return false, err
		}
		if fresh == nil {
			return false, nil
		}
		if provider.QoderRefreshCredentialsHash(fresh.Credentials) != failedCredentialsHash {
			if s.Tokens != nil {
				s.Tokens.InvalidateProvider(fresh)
			}
			result = fresh
			return true, nil
		}
		return false, nil
	})
	return result, err
}

type qoderFailedRefreshExecutor struct {
	*QoderTokenRefresher
	failedCredentials string
}

func (e qoderFailedRefreshExecutor) NeedsRefresh(value *provider.Record, ttl time.Duration) bool {
	if e.QoderTokenRefresher == nil {
		return false
	}
	return provider.NeedsRefreshQoderAfterFailure(value, e.failedCredentials, ttl)
}
