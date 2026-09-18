//go:build integration

package repository

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// 真实 Redis 验证跨实例租约和 TTL，使用独立合成键，不访问账号数据。
func TestCodexTicketCacheRealRedis(t *testing.T) {
	ctx := context.Background()
	key := uniqueTestValue(t, "ticket")
	t.Cleanup(func() {
		_ = integrationRedis.Del(ctx, "private:codex-ticket:v1:"+key, "private:codex-ticket-lease:v1:"+key, "private:codex-ticket-lease:v1:round:"+key).Err()
	})
	a, b := NewCodexTicketCache(integrationRedis), NewCodexTicketCache(integrationRedis)
	require.NoError(t, a.Set(ctx, key, "synthetic-encrypted-data", time.Minute))
	value, err := b.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, "synthetic-encrypted-data", value)
	values, err := b.GetMany(ctx, []string{key, key + "-missing"})
	require.NoError(t, err)
	require.Equal(t, map[string]string{key: "synthetic-encrypted-data"}, values)
	ttl, err := integrationRedis.TTL(ctx, "private:codex-ticket:v1:"+key).Result()
	require.NoError(t, err)
	require.Positive(t, ttl)
	require.LessOrEqual(t, ttl, time.Minute)
	ok, err := a.Claim(ctx, key, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = b.Claim(ctx, key, time.Minute)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = a.AcquireLease(ctx, "round:"+key, "owner-a", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, b.ReleaseLease(ctx, "round:"+key, "owner-b"))
	ok, err = b.RenewLease(ctx, "round:"+key, "owner-b", time.Minute)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = a.RenewLease(ctx, "round:"+key, "owner-a", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = b.AcquireLease(ctx, "round:"+key, "owner-b", time.Minute)
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, a.ReleaseLease(ctx, "round:"+key, "owner-a"))
	ok, err = b.AcquireLease(ctx, "round:"+key, "owner-b", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
}
