package selection

import (
	"context"
	"testing"
	"time"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	schedulercore "github.com/TokenFlux/TokenRouter/internal/scheduler"
	schedulerredis "github.com/TokenFlux/TokenRouter/internal/scheduler/rediscache"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func TestSparkRoutingByModel(t *testing.T) {
	ctx := context.Background()
	sparkModel := "gpt-5.3-codex-spark"
	normalModel := "gpt-5.3-codex"
	sparkCreds := map[string]any{"model_mapping": provideradapter.DefaultSparkShadowModels()}

	newScheduler := func(snapshot map[int64]*gatewayprovider.ExecutionProvider) *compatiblePicker {
		return &compatiblePicker{
			service: newCompatibleSelectionForTest(CompatibleDependencies{
				Reads: Reads{Snapshot: schedulerredis.NewSnapshotReader(schedulercore.NewSnapshotService(
					&openAISnapshotCacheStub{providersByID: snapshot}, nil, nil, nil, nil))},
				Shared: Shared{},
			}, &config.Config{}),
		}
	}
	sparkReq := schedulercore.PlatformSelectionInput{RequestedModel: sparkModel, Platform: capability.PlatformOpenAI}
	normalReq := schedulercore.PlatformSelectionInput{RequestedModel: normalModel, Platform: capability.PlatformOpenAI}

	t.Run("normal_provider_with_spark_mapping_accepts_spark", func(t *testing.T) {
		acc := &gatewayprovider.ExecutionProvider{Record: provider.Record{LoadLocation: time.LoadLocation, ID: 1, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive, Schedulable: true, Credentials: sparkCreds}}
		require.True(t, newScheduler(nil).isProviderRequestCompatible(ctx, acc, sparkReq),
			"普通提供商配了 spark → 可承接 spark（类型门已移除）")
	})

	t.Run("normal_provider_without_spark_rejects_spark", func(t *testing.T) {
		acc := &gatewayprovider.ExecutionProvider{Record: provider.Record{
			LoadLocation: time.LoadLocation, ID: 1, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive, Schedulable: true,
			Credentials: map[string]any{"model_whitelist": []string{normalModel}, "model_mapping": map[string]any{normalModel: normalModel}},
		}}
		require.False(t, newScheduler(nil).isProviderRequestCompatible(ctx, acc, sparkReq),
			"普通提供商未配 spark → 拒 spark（按配置而非类型）")
	})

	t.Run("shadow_with_spark_mapping_accepts_spark_rejects_non_spark", func(t *testing.T) {
		pid := int64(100)
		parent := &gatewayprovider.ExecutionProvider{Record: provider.Record{Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 100, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive, Schedulable: true}}
		shadow := &gatewayprovider.ExecutionProvider{Record: provider.Record{
			LoadLocation: time.LoadLocation, ID: 200, ParentProviderID: &pid, QuotaDimension: provider.QuotaDimensionSpark,
			Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive, Schedulable: true, Concurrency: 1, Credentials: sparkCreds,
		}}
		s := newScheduler(map[int64]*gatewayprovider.ExecutionProvider{100: parent})
		require.True(t, s.isProviderRequestCompatible(ctx, shadow, sparkReq), "影子配 spark + 健康母 → 接 spark")
		require.False(t, s.isProviderRequestCompatible(ctx, shadow, normalReq), "影子（仅 spark mapping）→ 拒非 spark")
	})

	t.Run("empty_model_shadow_is_eligible_under_a2", func(t *testing.T) {
		// 有意的纯 A2 行为(用户裁决 2026-06-30)：空 model 请求不经模型门过滤
		// （isProviderRequestCompatible 的 `req.RequestedModel != ""` 短路），故影子与普通提供商
		// 一样成为候选。旧类型门曾在空 model 时排除影子(opt-in)，该 opt-in 已随类型门移除——
		// routing 路径不再有任何类型判断。此测试锁定该决策，防被未来改动静默改回。
		pid := int64(100)
		parent := &gatewayprovider.ExecutionProvider{Record: provider.Record{Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 100, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive, Schedulable: true}}
		shadow := &gatewayprovider.ExecutionProvider{Record: provider.Record{
			LoadLocation: time.LoadLocation, ID: 200, ParentProviderID: &pid, QuotaDimension: provider.QuotaDimensionSpark,
			Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive, Schedulable: true, Concurrency: 1, Credentials: sparkCreds,
		}}
		emptyReq := schedulercore.PlatformSelectionInput{RequestedModel: "", Platform: capability.PlatformOpenAI}
		s := newScheduler(map[int64]*gatewayprovider.ExecutionProvider{100: parent})
		require.True(t, s.isProviderRequestCompatible(ctx, shadow, emptyReq),
			"空 model 时影子可被选中（有意的纯 A2 行为：类型门移除后无 opt-in 排除）")
	})
}

// TestParentHealthSchedulerIntegration 通过 isProviderRequestCompatible 验证「母提供商不可调度时影子被
// 调度器拒绝」这一联动在调度器层面端到端生效。
//
// 使用的接缝：defaultOpenAIProviderScheduler.isProviderRequestCompatible，它通过
// s.service.schedulerSnapshot.GetProvider(ctx, parentID) 解析母提供商；
// openAISnapshotCacheStub.providersByID 提供对应的测试桩。
func TestParentHealthSchedulerIntegration(t *testing.T) {
	ctx := context.Background()
	pid := int64(78100)
	sparkModel := "gpt-5.3-codex-spark"

	shadow := &gatewayprovider.ExecutionProvider{
		Record: provider.Record{
			Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 78200,
			ParentProviderID: &pid,
			QuotaDimension:   provider.QuotaDimensionSpark,
			Platform:         capability.PlatformOpenAI,
			Type:             capability.ProviderTypeOAuth,
			Status:           billing.StatusActive,
			Schedulable:      true,
			Concurrency:      1,
		},
	}

	req := schedulercore.PlatformSelectionInput{
		RequestedModel: sparkModel,
		Platform:       capability.PlatformOpenAI,
	}

	makeScheduler := func(parent *gatewayprovider.ExecutionProvider) *compatiblePicker {
		snapshotCache := &openAISnapshotCacheStub{
			providersByID: map[int64]*gatewayprovider.ExecutionProvider{parent.Record.ID: parent},
		}
		snapshotSvc := schedulercore.NewSnapshotService(snapshotCache, nil, nil, nil, nil)
		svc := newCompatibleSelectionForTest(CompatibleDependencies{
			Reads:  Reads{Snapshot: schedulerredis.NewSnapshotReader(snapshotSvc)},
			Shared: Shared{},
		}, &config.Config{})

		return &compatiblePicker{service: svc}
	}

	t.Run("unhealthy_parent_status_error_rejects_shadow", func(t *testing.T) {
		unhealthyParent := &gatewayprovider.ExecutionProvider{
			Record: provider.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 78100,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeOAuth,
				Status:      provider.StatusError, // IsActive()==false → IsSchedulable()==false
				Schedulable: true,
			},
		}
		require.False(t, unhealthyParent.View().IsSchedulable(), "前提：Status=error 的母提供商不可调度")
		scheduler := makeScheduler(unhealthyParent)
		require.False(t, scheduler.isProviderRequestCompatible(ctx, shadow, req),
			"母提供商不可调度时，影子提供商必须被调度器拒绝")
	})

	t.Run("manual_schedulable_false_parent_does_not_reject_shadow", func(t *testing.T) {
		// F1 决策 A:母提供商手动暂停(Schedulable=false)不传播到影子 —— 凭据仍可用,影子应被接受。
		manualPausedParent := &gatewayprovider.ExecutionProvider{
			Record: provider.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 78100,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeOAuth,
				Status:      billing.StatusActive,
				Schedulable: false,
			}, // 显式手动暂停
		}
		require.False(t, manualPausedParent.View().IsSchedulable(), "前提：手动暂停的母提供商自身不可调度")
		scheduler := makeScheduler(manualPausedParent)
		require.True(t, scheduler.isProviderRequestCompatible(ctx, shadow, req),
			"母提供商手动暂停不应连坐影子(凭据仍可用)")
	})

	t.Run("global_rate_limited_parent_does_not_reject_shadow", func(t *testing.T) {
		// F1 核心修复:母提供商 global 429(RateLimitResetAt)不连坐 spark 影子。
		resetAt := time.Now().Add(1 * time.Hour)
		rateLimitedParent := &gatewayprovider.ExecutionProvider{
			Record: provider.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 78100,
				Platform:         capability.PlatformOpenAI,
				Type:             capability.ProviderTypeOAuth,
				Status:           billing.StatusActive,
				Schedulable:      true,
				RateLimitResetAt: &resetAt,
			},
		}
		require.False(t, rateLimitedParent.View().IsSchedulable(), "前提：global 限流母提供商自身不可调度")
		scheduler := makeScheduler(rateLimitedParent)
		require.True(t, scheduler.isProviderRequestCompatible(ctx, shadow, req),
			"母提供商 global 限流不应连坐 spark 影子")
	})

	t.Run("healthy_parent_accepts_shadow_control", func(t *testing.T) {
		healthyParent := &gatewayprovider.ExecutionProvider{
			Record: provider.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 78100,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeOAuth,
				Status:      billing.StatusActive,
				Schedulable: true,
			},
		}
		require.True(t, healthyParent.View().IsSchedulable(), "前提：健康母提供商必须可调度")
		scheduler := makeScheduler(healthyParent)
		require.True(t, scheduler.isProviderRequestCompatible(ctx, shadow, req),
			"健康母提供商时，影子提供商必须被调度器接受（对照组）")
	})
}

func TestParentHealthSchedulerFallsBackToRepoWhenSnapshotMissesParent(t *testing.T) {
	ctx := context.Background()
	parentID := int64(79100)
	parent := gatewayprovider.ExecutionProvider{
		Record: provider.Record{
			Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: parentID,
			Platform:    capability.PlatformOpenAI,
			Type:        capability.ProviderTypeOAuth,
			Status:      billing.StatusActive,
			Schedulable: true,
		},
	}
	shadow := &gatewayprovider.ExecutionProvider{
		Record: provider.Record{
			Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 79200,
			ParentProviderID: &parentID,
			QuotaDimension:   provider.QuotaDimensionSpark,
			Platform:         capability.PlatformOpenAI,
			Type:             capability.ProviderTypeOAuth,
			Status:           billing.StatusActive,
			Schedulable:      true,
			Concurrency:      1,
		},
	}

	repo := schedulerTestOpenAIProviderRepo{providers: []gatewayprovider.ExecutionProvider{parent}}
	scheduler := &compatiblePicker{service: newCompatibleSelectionForTest(CompatibleDependencies{
		Reads: Reads{
			Providers: repo,
			Snapshot:  schedulerredis.NewSnapshotReader(schedulercore.NewSnapshotService(&openAISnapshotCacheStub{}, nil, hydrationProviderSource{source: repo}, nil, &schedulercore.SnapshotOptions{DbFallbackEnabled: false})),
		},
		Shared: Shared{},
	}, &config.Config{})}

	require.True(t, scheduler.isProviderRequestCompatible(ctx, shadow, schedulercore.PlatformSelectionInput{
		RequestedModel: "gpt-5.3-codex-spark",
		Platform:       capability.PlatformOpenAI,
	}), "快照缺失母提供商且调度快照 DB fallback 关闭时，应回退 repo 解析健康母提供商")
}
