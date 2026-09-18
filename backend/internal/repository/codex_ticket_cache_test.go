package repository

import (
	"context"
	"encoding/json"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCodexTicketLatestRejectsLateOlderWrite(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	c := NewCodexTicketCache(rdb)
	ctx := context.Background()
	require.NoError(t, c.SetLatest(ctx, "test", `{"finished_us":200,"source":"manual"}`, 200, time.Hour))
	require.NoError(t, c.SetLatest(ctx, "test", `{"finished_us":100,"source":"auto"}`, 100, time.Hour))
	raw, err := c.Get(ctx, "latest:test")
	require.NoError(t, err)
	var latest map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &latest))
	require.Equal(t, "manual", latest["source"])
	require.NoError(t, c.SetLatest(ctx, "test", `{"finished_us":300,"source":"auto"}`, 300, time.Hour))
	raw, err = c.Get(ctx, "latest:test")
	require.NoError(t, err)
	require.Contains(t, raw, `"auto"`)
}

// 合成采集使用独立 H1 短连接；默认业务传输仍允许连接复用。
func TestCodexTicketHarvestTransportProfileIsolation(t *testing.T) {
	svc := NewHTTPUpstream(&config.Config{}).(*httpUpstreamService)
	mode := svc.resolveProtocolMode(service.HTTPUpstreamProfileOpenAIHarvest, "proxy", nil)
	require.Equal(t, upstreamProtocolModeOpenAIH1NoReuse, mode)
	settings := poolSettings{maxIdleConns: 100, maxIdleConnsPerHost: 10}
	harvest, err := buildUpstreamTransport(settings, nil, mode)
	require.NoError(t, err)
	require.True(t, harvest.DisableKeepAlives)
	require.False(t, harvest.ForceAttemptHTTP2)
	require.NotNil(t, harvest.TLSNextProto)
	normal, err := buildUpstreamTransport(settings, nil, upstreamProtocolModeDefault)
	require.NoError(t, err)
	require.False(t, normal.DisableKeepAlives)
	require.Equal(t, 100, normal.MaxIdleConns)
}

// 缓存仅接收业务层加密后的内容，跨实例租约互斥且到期可恢复，不使用账号数据键。
func TestCodexTicketCacheTTLAndClusterLease(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	a, b := NewCodexTicketCache(rdb), NewCodexTicketCache(rdb)
	ctx := context.Background()
	value, err := a.Get(ctx, "missing")
	require.NoError(t, err)
	require.Empty(t, value)
	require.NoError(t, a.Set(ctx, "hashed-key", "encrypted-ticket", time.Hour))
	value, err = b.Get(ctx, "hashed-key")
	require.NoError(t, err)
	require.Equal(t, "encrypted-ticket", value)
	values, err := b.GetMany(ctx, []string{"hashed-key", "missing"})
	require.NoError(t, err)
	require.Equal(t, map[string]string{"hashed-key": "encrypted-ticket"}, values)
	ok, err := a.Claim(ctx, "round:generation", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = b.Claim(ctx, "round:generation", time.Minute)
	require.NoError(t, err)
	require.False(t, ok)
	require.ElementsMatch(t, []string{"private:codex-ticket:v1:hashed-key", "private:codex-ticket-lease:v1:round:generation"}, mr.Keys())
	mr.FastForward(time.Minute)
	ok, err = b.Claim(ctx, "round:generation", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	mr.FastForward(time.Hour)
	value, err = b.Get(ctx, "hashed-key")
	require.NoError(t, err)
	require.Empty(t, value)
}

// 新 owner 已接管过期租约后，旧 owner 的迟到释放不得删除新锁。
func TestCodexTicketLeaseOwnerRelease(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	c := NewCodexTicketCache(rdb)
	ctx := context.Background()
	ok, err := c.AcquireLease(ctx, "round", "a", time.Second)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, c.ReleaseLease(ctx, "round", "wrong"))
	ok, err = c.AcquireLease(ctx, "round", "b", time.Second)
	require.NoError(t, err)
	require.False(t, ok)
	mr.FastForward(2 * time.Second)
	ok, err = c.AcquireLease(ctx, "round", "b", time.Second)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, c.ReleaseLease(ctx, "round", "a"))
	owner, err := rdb.Get(ctx, "private:codex-ticket-lease:v1:round").Result()
	require.NoError(t, err)
	require.Equal(t, "b", owner)
	require.NoError(t, c.ReleaseLease(ctx, "round", "b"))
	ok, err = c.AcquireLease(ctx, "round", "c", time.Second)
	require.NoError(t, err)
	require.True(t, ok)
}
