package repository

import (
	"context"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

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
