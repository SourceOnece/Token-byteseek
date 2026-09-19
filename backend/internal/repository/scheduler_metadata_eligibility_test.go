//go:build unit

package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/stretchr/testify/require"
)

// 通过生产Redis投影验证资格一致，避免只用完整账号夹具遗漏初筛路径。
func TestSchedulerMetadataPreservesEligibility(t *testing.T) {
	now := time.Now().UTC()
	full := service.Account{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth,
		Status: service.StatusActive, Schedulable: true,
		Credentials: map[string]any{"auth_mode": service.OpenAIAuthModeAgentIdentity, "account_scheduling_threshold": 100,
			"access_token": "synthetic-secret", "refresh_token": "synthetic-refresh", "private_key": "synthetic-private"},
		Extra: map[string]any{"privacy_mode": service.PrivacyModeTrainingOff, "codex_7d_used_percent": 90.0,
			"codex_7d_reset_at": now.Add(time.Hour).Format(time.RFC3339), "codex_usage_updated_at": now.Format(time.RFC3339),
			"openai_compact_mode": "force_off", "openai_native_compaction_v2_supported": false}}
	cache := newSchedulerCacheUnit(t)
	ctx := context.Background()
	bucket := service.SchedulerBucket{Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	token, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	require.NoError(t, cache.SetSnapshot(ctx, bucket, token, []service.Account{full}))
	accounts, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, accounts, 1)
	meta := accounts[0]
	require.True(t, meta.IsOpenAIAgentIdentity())
	require.True(t, meta.IsPrivacySet())
	require.False(t, meta.AllowsOpenAICompact())
	require.False(t, meta.AllowsOpenAINativeCompactionV2())
	limits := map[string]int{service.PlatformOpenAI: 80}
	require.Equal(t, service.EvaluateAccountSchedulingThreshold(&full, limits, now), service.EvaluateAccountSchedulingThreshold(meta, limits, now))
	require.False(t, service.EvaluateAccountSchedulingThreshold(meta, limits, now).ShouldPause)
	for _, key := range []string{"access_token", "refresh_token", "private_key"} {
		require.NotContains(t, meta.Credentials, key)
	}
	for _, key := range []string{"openai_compact_supported", "openai_native_compaction_v2_mode", "session_window_utilization",
		"passive_usage_7d_utilization", "passive_usage_7d_reset", "passive_usage_7d_oi_utilization", "passive_usage_7d_oi_reset"} {
		full.Extra[key] = "test-value"
		require.Equal(t, full.Extra[key], buildSchedulerMetadataAccount(full).Extra[key], key)
	}
}

func TestSchedulerMetadataLegacyCacheReprojectsWithoutWriting(t *testing.T) {
	for _, mode := range []string{"legacy", "missing_full", "wrong_full_id", "current"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			cache := newSchedulerCacheUnit(t)
			full := service.Account{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth,
				Credentials: map[string]any{"auth_mode": service.OpenAIAuthModeAgentIdentity, "access_token": "synthetic"},
				Extra:       map[string]any{"privacy_mode": service.PrivacyModeTrainingOff}}
			bucket := service.SchedulerBucket{Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
			token, err := cache.CaptureBucketWriteToken(ctx, bucket)
			require.NoError(t, err)
			require.NoError(t, cache.SetSnapshot(ctx, bucket, token, []service.Account{full}))
			if mode != "current" {
				// 模拟上一版写出的不带版本、不含资格字段的摘要。
				old := full
				old.Credentials, old.Extra = nil, nil
				payload, err := json.Marshal(old)
				require.NoError(t, err)
				require.NoError(t, cache.rdb.Set(ctx, schedulerAccountMetaKey("1"), payload, 0).Err())
			}
			if mode == "missing_full" || mode == "current" {
				require.NoError(t, cache.rdb.Del(ctx, schedulerAccountKey("1")).Err())
			}
			if mode == "wrong_full_id" {
				full.ID = 2
				payload, err := json.Marshal(full)
				require.NoError(t, err)
				require.NoError(t, cache.rdb.Set(ctx, schedulerAccountKey("1"), payload, 0).Err())
			}
			before, err := cache.rdb.Get(ctx, schedulerAccountMetaKey("1")).Result()
			require.NoError(t, err)
			accounts, hit, err := cache.GetSnapshot(ctx, bucket)
			require.NoError(t, err)
			if mode == "missing_full" || mode == "wrong_full_id" {
				require.False(t, hit)
				require.Nil(t, accounts)
			} else {
				require.True(t, hit)
				require.Len(t, accounts, 1)
				require.True(t, accounts[0].IsOpenAIAgentIdentity())
				require.True(t, accounts[0].IsPrivacySet())
				require.NotContains(t, accounts[0].Credentials, "access_token")
			}
			after, err := cache.rdb.Get(ctx, schedulerAccountMetaKey("1")).Result()
			require.NoError(t, err)
			require.Equal(t, before, after, "读修复不回写，避免覆盖并发更新")
		})
	}
}

func TestSchedulerMetadataThresholdIdentityAndOtherWindows(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	reset := now.Add(time.Hour)
	for _, platform := range []string{service.PlatformOpenAI, service.PlatformAnthropic, service.PlatformGrok} {
		for _, override := range []int{80, 100} {
			full := service.Account{Platform: platform, Type: service.AccountTypeOAuth, SessionWindowEnd: &reset,
				Credentials: map[string]any{"account_scheduling_threshold": override}, Extra: map[string]any{
					"codex_7d_used_percent": 90.0, "codex_7d_reset_at": reset.Format(time.RFC3339), "codex_usage_updated_at": now.Format(time.RFC3339),
					"session_window_utilization": 0.9, "passive_usage_7d_utilization": 0.9, "passive_usage_7d_reset": reset.Format(time.RFC3339),
					"grok_sched_utilization": 90.0, "grok_sched_reset_at": reset.Format(time.RFC3339)}}
			meta := buildSchedulerMetadataAccount(full)
			limits := map[string]int{platform: 50}
			want := service.EvaluateAccountSchedulingThreshold(&full, limits, now)
			require.Equal(t, override < 100, want.ShouldPause, platform)
			require.Equal(t, want, service.EvaluateAccountSchedulingThreshold(&meta, limits, now), platform)
			if platform == service.PlatformOpenAI {
				// 切换工作区后旧额度观测不可信，完整账号和摘要必须同样忽略。
				full.Credentials["chatgpt_account_id"] = "current-workspace"
				full.Extra["chatgpt_account_id"] = "previous-workspace"
				meta = buildSchedulerMetadataAccount(full)
				require.False(t, service.EvaluateAccountSchedulingThreshold(&meta, limits, now).ShouldPause)
			}
		}
	}
}
