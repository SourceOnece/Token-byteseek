package selection

import (
	"context"
	"errors"
	"testing"
	"time"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

// sessionLimitReleaseCacheStub 记录 UnregisterSession 调用，用于验证释放逻辑
type sessionLimitReleaseCacheStub struct {
	scheduler.SessionLimitCache

	unregistered map[int64][]string
	err          error
}

func newSessionLimitReleaseCacheStub() *sessionLimitReleaseCacheStub {
	return &sessionLimitReleaseCacheStub{
		unregistered: make(map[int64][]string),
	}
}

func (s *sessionLimitReleaseCacheStub) UnregisterSession(_ context.Context, providerID int64, sessionUUID string) error {
	if s.err != nil {
		return s.err
	}
	s.unregistered[providerID] = append(s.unregistered[providerID], sessionUUID)
	return nil
}

func newSessionLimitTestProvider() *gatewayprovider.ExecutionProvider {
	return &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, ID: 42,
			Platform: capability.PlatformAnthropic,
			Type:     capability.ProviderTypeOAuth,
			Extra:    map[string]any{"max_sessions": 1},
		},
	}
}

// TestReleaseProviderSession_ReleasesRegisteredSlot 验证：
// 对启用会话限制的 Anthropic OAuth 提供商，ReleaseProviderSession 必须立即移除
// 该提供商上注册的会话（不等待空闲超时）。
func TestReleaseProviderSession_ReleasesRegisteredSlot(t *testing.T) {
	cache := newSessionLimitReleaseCacheStub()
	svc := newGenericSelectionForTest(GenericDependencies{
		Reads: Reads{}, Shared: Shared{}, Sessions: cache,
	}, nil)

	acc := newSessionLimitTestProvider()

	svc.ReleaseProviderSession(context.Background(), acc, "session-hash-1")

	require.Equal(t, []string{"session-hash-1"}, cache.unregistered[42],
		"应立即移除注册的会话槽")
}

// TestReleaseProviderSession_NoOpForInapplicableProviders 验证：
// - 非 Anthropic OAuth/SetupToken 提供商
// - 未启用 max_sessions 的提供商
// - 空 sessionID
// 以上场景均为 no-op，不得触发 UnregisterSession。
func TestReleaseProviderSession_NoOpForInapplicableProviders(t *testing.T) {
	apiKeyAcc := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 43,
			Platform: capability.PlatformAnthropic,
			Type:     capability.ProviderTypeAPIKey,
			Extra:    map[string]any{"max_sessions": 1},
		},
	}
	noLimitAcc := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 44,
			Platform: capability.PlatformAnthropic,
			Type:     capability.ProviderTypeOAuth,
		},
	}
	enabledAcc := newSessionLimitTestProvider()

	cases := []struct {
		name      string
		provider  *gatewayprovider.ExecutionProvider
		sessionID string
	}{
		{"api_key_provider", apiKeyAcc, "session-hash"},
		{"max_sessions_disabled", noLimitAcc, "session-hash"},
		{"empty_session_id", enabledAcc, ""},
		{"nil_provider", nil, "session-hash"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cache := newSessionLimitReleaseCacheStub()
			svc := newGenericSelectionForTest(GenericDependencies{
				Reads: Reads{}, Shared: Shared{}, Sessions: cache,
			}, nil)

			svc.ReleaseProviderSession(context.Background(), tc.provider, tc.sessionID)
			require.Empty(t, cache.unregistered, "不适用提供商不应触发释放")
		})
	}
}

// TestReleaseProviderSession_NilCacheAndErrorTolerance 验证：
// sessionLimitCache 不可用时 no-op；UnregisterSession 返回错误时不 panic（仅记录日志）。
func TestReleaseProviderSession_NilCacheAndErrorTolerance(t *testing.T) {
	// nil cache：no-op
	svc := newGenericSelectionForTest(GenericDependencies{
		Reads: Reads{}, Shared: Shared{},
	}, nil)

	svc.ReleaseProviderSession(context.Background(), newSessionLimitTestProvider(), "session-hash")

	// 底层错误：不 panic
	cache := &sessionLimitReleaseCacheStub{err: errors.New("redis down")}
	svc = newGenericSelectionForTest(GenericDependencies{
		Reads: Reads{}, Shared: Shared{}, Sessions: cache,
	}, nil)

	svc.ReleaseProviderSession(context.Background(), newSessionLimitTestProvider(), "session-hash")
}

// TestReleaseProviderSession_Idempotent 验证释放操作幂等，可安全重复调用
// （failover 链上按次释放 + defer 兜底可能对同一提供商重复释放）。
func TestReleaseProviderSession_Idempotent(t *testing.T) {
	cache := newSessionLimitReleaseCacheStub()
	svc := newGenericSelectionForTest(GenericDependencies{
		Reads: Reads{}, Shared: Shared{}, Sessions: cache,
	}, nil)

	acc := newSessionLimitTestProvider()

	svc.ReleaseProviderSession(context.Background(), acc, "session-hash")
	svc.ReleaseProviderSession(context.Background(), acc, "session-hash")

	// 两次调用都透传到缓存层（Redis ZREM 本身幂等，重复移除无副作用）
	require.Len(t, cache.unregistered[42], 2)
}
