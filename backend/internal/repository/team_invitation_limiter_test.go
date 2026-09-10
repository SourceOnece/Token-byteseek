//go:build unit

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestTeamInvitationLimiterRecipientCooldown(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	limiter := NewTeamInvitationLimiter(client)

	allowed, retryAfter, err := limiter.CheckAndRecord(context.Background(), 11, " Member@Example.com ", service.DefaultTeamInvitationRateLimits())
	require.NoError(t, err)
	require.True(t, allowed)
	require.Zero(t, retryAfter)

	allowed, retryAfter, err = limiter.CheckAndRecord(context.Background(), 11, "member@example.com", service.DefaultTeamInvitationRateLimits())
	require.NoError(t, err)
	require.False(t, allowed)
	require.Greater(t, retryAfter, time.Duration(0))
	require.LessOrEqual(t, retryAfter, time.Duration(service.DefaultTeamInvitationCooldownSeconds)*time.Second)
}

func TestTeamInvitationLimiterHourlyLimit(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	limiter := NewTeamInvitationLimiter(client)
	ctx := context.Background()

	for index := 0; index < service.DefaultTeamInvitationHourlyLimit; index++ {
		allowed, retryAfter, err := limiter.CheckAndRecord(ctx, 22, fmt.Sprintf("member-%d@example.com", index), service.DefaultTeamInvitationRateLimits())
		require.NoError(t, err)
		require.True(t, allowed)
		require.Zero(t, retryAfter)
	}

	allowed, retryAfter, err := limiter.CheckAndRecord(ctx, 22, "overflow@example.com", service.DefaultTeamInvitationRateLimits())
	require.NoError(t, err)
	require.False(t, allowed)
	require.Greater(t, retryAfter, time.Duration(0))
	require.LessOrEqual(t, retryAfter, teamInvitationHourlyWindow)
}

func TestTeamInvitationLimiterCustomLimitsAndUpdates(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	l := NewTeamInvitationLimiter(client)
	ctx := context.Background()
	limits := service.TeamInvitationRateLimits{CooldownSeconds: 5, HourlyLimit: 2}
	a, _, err := l.CheckAndRecord(ctx, 1, "one@example.test", limits)
	require.NoError(t, err)
	require.True(t, a)
	server.FastForward(2 * time.Second)
	// 缩短配置不会取消已经进入的旧冷却；未到期时不消耗新次数。
	limits.CooldownSeconds = 1
	a, retry, err := l.CheckAndRecord(ctx, 1, "ONE@example.test", limits)
	require.NoError(t, err)
	require.False(t, a)
	require.Equal(t, 3*time.Second, retry)
	server.FastForward(3 * time.Second)
	a, _, err = l.CheckAndRecord(ctx, 1, "one@example.test", limits)
	require.NoError(t, err)
	require.True(t, a)
	a, _, err = l.CheckAndRecord(ctx, 1, "two@example.test", limits)
	require.NoError(t, err)
	require.False(t, a)
	// 增加上限沿用小时次数和窗口，不重新从零计数。
	limits.HourlyLimit = 3
	a, _, err = l.CheckAndRecord(ctx, 1, "two@example.test", limits)
	require.NoError(t, err)
	require.True(t, a)
	a, _, err = l.CheckAndRecord(ctx, 1, "three@example.test", limits)
	require.NoError(t, err)
	require.False(t, a)
	count, err := server.Get("team:invite:1:hourly")
	require.NoError(t, err)
	require.Equal(t, "3", count)
	require.Equal(t, time.Hour-5*time.Second, server.TTL("team:invite:1:hourly"))
	a, _, err = l.CheckAndRecord(ctx, 2, "one@example.test", limits)
	require.NoError(t, err)
	require.True(t, a)
	limits.HourlyLimit = 0
	a, _, err = l.CheckAndRecord(ctx, 1, "invalid@example.test", limits)
	require.Error(t, err)
	require.False(t, a)
}
