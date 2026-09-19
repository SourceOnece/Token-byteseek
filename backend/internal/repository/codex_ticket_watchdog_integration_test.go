//go:build integration

package repository

import (
	"context"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// 真实Redis核对跨实例CAS废票、重复观察与旧请求不能废新票。
func TestCodexTicketWatchdogAtomicRedis(t *testing.T) {
	ctx := context.Background()
	rdb := testRedis(t)
	a := NewCodexTicketCache(rdb)
	b := NewCodexTicketCache(rdb).(service.CodexTicketWatchdogCache)
	key := "synthetic-ticket"
	old := "cipher-old"
	newer := "cipher-new"
	require.NoError(t, a.Set(ctx, key, old, time.Hour))
	changed, err := b.ObserveTicket(ctx, key, old, "model_mismatch", false, time.Now())
	require.NoError(t, err)
	require.True(t, changed)
	got, err := a.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, old, got)
	changed, err = b.ObserveTicket(ctx, key, old, "model_mismatch", false, time.Now())
	require.NoError(t, err)
	require.False(t, changed)
	require.NoError(t, a.Set(ctx, key, newer, time.Hour))
	changed, err = b.ObserveTicket(ctx, key, old, "length_signal", true, time.Now())
	require.NoError(t, err)
	require.False(t, changed)
	got, err = a.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, newer, got)
	changed, err = b.ObserveTicket(ctx, key, newer, "length_signal", true, time.Now())
	require.NoError(t, err)
	require.True(t, changed)
	got, err = a.Get(ctx, key)
	require.NoError(t, err)
	require.Empty(t, got)
	state, err := a.Get(ctx, "watchdog:"+key)
	require.NoError(t, err)
	require.Contains(t, state, `"revoked"`)
	require.NotContains(t, state, "cipher-new")
}

func TestCodexTicketSettingsDatabaseCAS(t *testing.T) {
	ctx := context.Background()
	repo := NewSettingRepository(testEntClient(t)).(*settingRepository)
	const key = "codex_ticket_runtime"
	// 集成数据库独立于生产，移除本测试创建的设置，避免影响同包其他用例。
	t.Cleanup(func() { _ = repo.Delete(ctx, key) })
	current, err := repo.GetValue(ctx, key)
	var expected *string
	if err == nil {
		expected = &current
	}
	ok, err := repo.CompareAndSwapTicketSettings(ctx, expected, `{"generation":"one"}`)
	require.NoError(t, err)
	require.True(t, ok)
	old := `{"generation":"one"}`
	ok, err = repo.CompareAndSwapTicketSettings(ctx, &old, `{"generation":"two"}`)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.CompareAndSwapTicketSettings(ctx, &old, `{"generation":"stale"}`)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = repo.CompareAndSwapTicketSettings(ctx, nil, `{"generation":"create-conflict"}`)
	require.NoError(t, err)
	require.False(t, ok)
	got, err := repo.GetValue(ctx, key)
	require.NoError(t, err)
	require.Equal(t, `{"generation":"two"}`, got)
}
