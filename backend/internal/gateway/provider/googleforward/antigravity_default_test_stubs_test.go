//go:build !unit

package googleforward_test

import (
	"context"
	"time"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
)

type defaultRateLimitCall struct {
	providerID int64
	resetAt    time.Time
}

type defaultModelRateLimitCall struct {
	providerID int64
	modelKey   string
	resetAt    time.Time
}

type defaultExtraUpdateCall struct {
	providerID int64
	updates    map[string]any
}

type stubAntigravityProviderRepo struct {
	gatewayprovider.ExecutionProviderStore

	rateCalls           []defaultRateLimitCall
	modelRateLimitCalls []defaultModelRateLimitCall
	extraUpdateCalls    []defaultExtraUpdateCall
}

func (s *stubAntigravityProviderRepo) SetRateLimited(_ context.Context, id int64, resetAt time.Time) error {
	s.rateCalls = append(s.rateCalls, defaultRateLimitCall{providerID: id, resetAt: resetAt})
	return nil
}

func (s *stubAntigravityProviderRepo) SetModelRateLimit(_ context.Context, id int64, modelKey string, resetAt time.Time, _ ...string) error {
	s.modelRateLimitCalls = append(s.modelRateLimitCalls, defaultModelRateLimitCall{providerID: id, modelKey: modelKey, resetAt: resetAt})
	return nil
}

func (s *stubAntigravityProviderRepo) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	s.extraUpdateCalls = append(s.extraUpdateCalls, defaultExtraUpdateCall{providerID: id, updates: updates})
	return nil
}

type defaultDeleteSessionCall struct {
	groupID     int64
	sessionHash string
}

type stubSmartRetryCache struct {
	session.GatewayCache
	deleteCalls []defaultDeleteSessionCall
}

func (c *stubSmartRetryCache) DeleteSessionProviderID(_ context.Context, groupID int64, sessionHash string) error {
	c.deleteCalls = append(c.deleteCalls, defaultDeleteSessionCall{groupID: groupID, sessionHash: sessionHash})
	return nil
}
