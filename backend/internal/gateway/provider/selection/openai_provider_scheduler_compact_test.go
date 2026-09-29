package selection

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	gatewaytestkit "github.com/TokenFlux/TokenRouter/internal/gateway/testkit"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/infra/telemetry/logging"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	schedulerredis "github.com/TokenFlux/TokenRouter/internal/scheduler/rediscache"
	"github.com/stretchr/testify/require"
)

// 新旧调度器都必须复核数据库开关，缓存显示关闭也不能永久漏选已重新启用的提供商。
func TestCompactSchedulingRechecksAdministratorSwitchFromDatabase(t *testing.T) {
	for _, advanced := range []bool{false, true} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("advanced=%v/enabled=%v", advanced, enabled), func(t *testing.T) {
				groupID := int64(91090)
				cachedMode, dbMode := "force_on", "force_off"
				if enabled {
					cachedMode, dbMode = dbMode, cachedMode
				}
				cached := &gatewayprovider.ExecutionProvider{Record: providercore.Record{
					Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 71990, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive,
					Schedulable: true, Concurrency: 1, GroupIDs: []int64{groupID}, Extra: map[string]any{"openai_compact_mode": cachedMode},
				}}
				fresh := *cached
				fresh.Record.Extra = map[string]any{"openai_compact_mode": dbMode}
				svc := newCompatibleSelectionForTest(CompatibleDependencies{
					Reads: Reads{
						Providers: schedulerTestOpenAIProviderRepo{providers: []gatewayprovider.ExecutionProvider{fresh}},
						Snapshot:  schedulerredis.NewSnapshotReader(scheduler.NewSnapshotService(&openAISnapshotCacheStub{snapshotProviders: []*gatewayprovider.ExecutionProvider{cached}, providersByID: map[int64]*gatewayprovider.ExecutionProvider{cached.Record.ID: cached}}, nil, nil, nil, nil)),
					},
					Shared: Shared{
						Cache:       &schedulerTestGatewayCache{},
						Concurrency: scheduler.NewConcurrencyService(schedulerTestConcurrencyCache{}, scheduler.Diagnostics{Logf: logging.LegacyPrintf, Event: logging.Event}),

						Health:     gatewaytestkit.NewHealthObserver(gatewaytestkit.HealthInput{}),
						Parameters: newAdvancedSchedulerParametersForTest(&config.Config{}, fmt.Sprint(advanced)),
					},
				}, &config.Config{})

				selected, _, err := svc.SelectProviderWithScheduler(context.Background(), &groupID, "", "", "gpt-5.4", nil, egress.OpenAIUpstreamTransportAny, true)
				if enabled {
					require.NoError(t, err)
					require.NotNil(t, selected)
					require.Equal(t, cached.Record.ID, selected.Provider.Record.ID)
				} else {
					require.ErrorIs(t, err, scheduler.ErrNoAvailableCompactProviders)
					require.Nil(t, selected)
				}
			})
		}
	}
}

// TestOpenAIGatewayService_SelectProviderWithScheduler_CompactSelectsEnabledProvider
// 验证 Compact 调度只选择管理员启用的提供商。
func TestOpenAIGatewayService_SelectProviderWithScheduler_CompactSelectsEnabledProvider(t *testing.T) {
	ctx := context.Background()
	groupID := int64(91001)
	providers := []gatewayprovider.ExecutionProvider{
		{
			Record: providercore.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 71001,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeAPIKey,
				Status:      billing.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    0,
				Extra:       map[string]any{"openai_compact_mode": "force_off"},
			}, // 管理员禁用
		},
		{
			Record: providercore.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 71002,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeAPIKey,
				Status:      billing.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    0,
				Extra:       map[string]any{"openai_compact_mode": "force_on"},
			}, // 管理员启用
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := newCompatibleSelectionForTest(CompatibleDependencies{
		Reads: Reads{
			Providers: schedulerTestOpenAIProviderRepo{providers: providers},
		},
		Shared: Shared{
			Cache:       &schedulerTestGatewayCache{},
			Concurrency: scheduler.NewConcurrencyService(schedulerTestConcurrencyCache{}, scheduler.Diagnostics{Logf: logging.LegacyPrintf, Event: logging.Event}),
		},
	}, cfg)

	selection, _, err := svc.SelectProviderWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.4",
		nil, egress.OpenAIUpstreamTransportAny, true,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Provider)
	require.Equal(t, int64(71002), selection.Provider.Record.ID, "disabled provider must be excluded")
}

// TestOpenAIGatewayService_SelectProviderWithScheduler_CompactRejectsDisabled
// 验证管理员关闭的提供商不会被 Compact 请求选中。
func TestOpenAIGatewayService_SelectProviderWithScheduler_CompactRejectsDisabled(t *testing.T) {
	ctx := context.Background()
	groupID := int64(91002)
	providers := []gatewayprovider.ExecutionProvider{
		{
			Record: providercore.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 71010,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeAPIKey,
				Status:      billing.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    0,
				Extra:       map[string]any{"openai_compact_mode": providercore.OpenAICompactModeForceOff},
			},
		},
		{
			Record: providercore.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 71011,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeAPIKey,
				Status:      billing.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    0,
				Extra:       map[string]any{"openai_compact_mode": "force_off"},
			},
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := newCompatibleSelectionForTest(CompatibleDependencies{
		Reads: Reads{
			Providers: schedulerTestOpenAIProviderRepo{providers: providers},
		},
		Shared: Shared{
			Cache:       &schedulerTestGatewayCache{},
			Concurrency: scheduler.NewConcurrencyService(schedulerTestConcurrencyCache{}, scheduler.Diagnostics{Logf: logging.LegacyPrintf, Event: logging.Event}),
		},
	}, cfg)

	selection, _, err := svc.SelectProviderWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.4",
		nil, egress.OpenAIUpstreamTransportAny, true,
	)
	require.Error(t, err)
	require.True(t, errors.Is(err, scheduler.ErrNoAvailableCompactProviders), "compact-only providers should rejected explicitly unsupported and return compact error")
	require.Nil(t, selection)
}

// TestOpenAIGatewayService_SelectProviderWithScheduler_CompactUsesDefaultEnabledProvider
// 验证缺省压缩开关保持开启，显式关闭仍然排除。
func TestOpenAIGatewayService_SelectProviderWithScheduler_CompactUsesDefaultEnabledProvider(t *testing.T) {
	ctx := context.Background()
	groupID := int64(91003)
	providers := []gatewayprovider.ExecutionProvider{
		{
			Record: providercore.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 71020,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeAPIKey,
				Status:      billing.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    0,
				Extra:       map[string]any{"openai_compact_mode": "force_off"},
			}, // 管理员关闭
		},
		{
			Record: providercore.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 71021,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeAPIKey,
				Status:      billing.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    0,
				Extra:       map[string]any{},
			}, // 缺省开启
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := newCompatibleSelectionForTest(CompatibleDependencies{
		Reads: Reads{
			Providers: schedulerTestOpenAIProviderRepo{providers: providers},
		},
		Shared: Shared{
			Cache:       &schedulerTestGatewayCache{},
			Concurrency: scheduler.NewConcurrencyService(schedulerTestConcurrencyCache{}, scheduler.Diagnostics{Logf: logging.LegacyPrintf, Event: logging.Event}),
		},
	}, cfg)

	selection, _, err := svc.SelectProviderWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.4",
		nil, egress.OpenAIUpstreamTransportAny, true,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Provider)
	require.Equal(t, int64(71021), selection.Provider.Record.ID, "default-enabled provider should be selected")
}

// TestOpenAIGatewayService_SelectProviderWithScheduler_CompactAllowsGrok 验证 compact 调度允许 Grok 提供商参与。
func TestOpenAIGatewayService_SelectProviderWithScheduler_CompactAllowsGrok(t *testing.T) {
	ctx := context.Background()
	groupID := int64(91004)
	providers := []gatewayprovider.ExecutionProvider{
		{
			Record: providercore.Record{
				LoadLocation: time.LoadLocation, ID: 71030,
				Platform:    capability.PlatformGrok,
				Type:        capability.ProviderTypeOAuth,
				Status:      billing.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    0,
				Credentials: map[string]any{
					"model_mapping": map[string]any{"grok-4.5": "grok-4.5"},
				},
			},
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := newCompatibleSelectionForTest(CompatibleDependencies{
		Reads: Reads{
			Providers: schedulerTestOpenAIProviderRepo{providers: providers},
		},
		Shared: Shared{
			Cache:       &schedulerTestGatewayCache{},
			Concurrency: scheduler.NewConcurrencyService(schedulerTestConcurrencyCache{}, scheduler.Diagnostics{Logf: logging.LegacyPrintf, Event: logging.Event}),
		},
	}, cfg)

	selection, _, err := svc.SelectProviderWithSchedulerForCapability(
		ctx,
		&groupID,
		"",
		"",
		"grok-4.5",
		nil, egress.OpenAIUpstreamTransportAny, providercore.OpenAIEndpointCapabilityTextGeneration,
		true,
		false,
		capability.PlatformGrok,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Provider)
	require.Equal(t, int64(71030), selection.Provider.Record.ID)
}

func TestOpenAIGatewayService_SelectProviderWithScheduler_NativeCompactionSeparatesLegacyCompactSupport(t *testing.T) {
	groupID := int64(91005)
	providers := []gatewayprovider.ExecutionProvider{
		{
			Record: providercore.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 71050,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeAPIKey,
				Status:      billing.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    10,
				Extra: map[string]any{
					"openai_compact_mode":    "force_on",
					"openai_text_route_mode": "force_chat_completions",
				},
			},
		},
		{
			Record: providercore.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 71051,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeAPIKey,
				Status:      billing.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Extra: map[string]any{
					"openai_compact_mode": providercore.OpenAICompactModeForceOff,
				},
			},
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := newCompatibleSelectionForTest(CompatibleDependencies{
		Reads: Reads{
			Providers: schedulerTestOpenAIProviderRepo{providers: providers},
		},
		Shared: Shared{
			Cache:       &schedulerTestGatewayCache{},
			Concurrency: scheduler.NewConcurrencyService(schedulerTestConcurrencyCache{}, scheduler.Diagnostics{Logf: logging.LegacyPrintf, Event: logging.Event}),
		},
	}, cfg)

	nativeSelection, _, err := svc.SelectProviderWithSchedulerForCapability(
		context.Background(),
		&groupID,
		"",
		"",
		"gpt-5.6-sol",
		nil, egress.OpenAIUpstreamTransportAny, providercore.OpenAIEndpointCapabilityResponses,
		false,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, nativeSelection)
	require.Equal(t, int64(71051), nativeSelection.Provider.Record.ID)

	legacySelection, _, err := svc.SelectProviderWithSchedulerForCapability(
		context.Background(),
		&groupID,
		"",
		"",
		"gpt-5.6-sol",
		nil, egress.OpenAIUpstreamTransportAny, providercore.OpenAIEndpointCapabilityResponses,
		true,
		false,
	)
	require.ErrorIs(t, err, scheduler.ErrNoAvailableCompactProviders)
	require.Nil(t, legacySelection)
}

// TestOpenAIGatewayService_SelectProviderWithScheduler_NativeCompactionV2Mode
// 验证原生 V2 只读取自身管理员开关，忽略历史状态和旧版 Compact 开关。
func TestOpenAIGatewayService_SelectProviderWithScheduler_NativeCompactionV2Mode(t *testing.T) {
	groupID := int64(91006)
	providers := []gatewayprovider.ExecutionProvider{
		{
			Record: providercore.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 71060,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeAPIKey,
				Status:      billing.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    0,
				Extra: map[string]any{
					providercore.OpenAINativeCompactionV2ModeExtraKey: providercore.OpenAICompactModeForceOff,
					"openai_native_compaction_v2_supported":           true,
				},
			},
		},
		{
			Record: providercore.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 71061,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeAPIKey,
				Status:      billing.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				Extra: map[string]any{
					"openai_native_compaction_v2_supported":           false,
					providercore.OpenAINativeCompactionV2ModeExtraKey: providercore.OpenAICompactModeForceOff,
				},
			},
		},
		{
			Record: providercore.Record{
				Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 71062,
				Platform:    capability.PlatformOpenAI,
				Type:        capability.ProviderTypeAPIKey,
				Status:      billing.StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    2,
				Extra: map[string]any{
					providercore.OpenAINativeCompactionV2ModeExtraKey: providercore.OpenAICompactModeForceOn,
					"openai_native_compaction_v2_supported":           false,
				},
			},
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := newCompatibleSelectionForTest(CompatibleDependencies{
		Reads: Reads{
			Providers: schedulerTestOpenAIProviderRepo{providers: providers},
		},
		Shared: Shared{
			Cache:       &schedulerTestGatewayCache{},
			Concurrency: scheduler.NewConcurrencyService(schedulerTestConcurrencyCache{}, scheduler.Diagnostics{Logf: logging.LegacyPrintf, Event: logging.Event}),
		},
	}, cfg)

	selection, _, err := svc.SelectProviderWithSchedulerForCapability(
		context.Background(),
		&groupID,
		"",
		"",
		"gpt-5.6-sol",
		nil, egress.OpenAIUpstreamTransportAny, providercore.OpenAIEndpointCapabilityRemoteCompactionV2,
		false,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Provider)
	require.Equal(t, int64(71062), selection.Provider.Record.ID)
}

// TestAllowsOpenAICompatibleCompact 验证压缩资格开关。
func TestAllowsOpenAICompatibleCompact(t *testing.T) {
	tests := []struct {
		name     string
		provider *gatewayprovider.ExecutionProvider
		want     bool
	}{
		{name: "nil", provider: nil, want: false},
		{name: "non openai", provider: &gatewayprovider.ExecutionProvider{Record: providercore.Record{Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic}}, want: false},
		{name: "grok", provider: &gatewayprovider.ExecutionProvider{Record: providercore.Record{Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, Platform: capability.PlatformGrok}}, want: true},
		{name: "openai default enabled", provider: &gatewayprovider.ExecutionProvider{Record: providercore.Record{Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Extra: map[string]any{}}}, want: true},
		{name: "openai enabled", provider: &gatewayprovider.ExecutionProvider{Record: providercore.Record{Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Extra: map[string]any{"openai_compact_mode": "force_on"}}}, want: true},
		{name: "openai disabled", provider: &gatewayprovider.ExecutionProvider{Record: providercore.Record{Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Extra: map[string]any{"openai_compact_mode": "force_off"}}}, want: false},
		{name: "force on", provider: &gatewayprovider.ExecutionProvider{Record: providercore.Record{Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Extra: map[string]any{"openai_compact_mode": providercore.OpenAICompactModeForceOn}}}, want: true},
		{name: "force off overrides probe true", provider: &gatewayprovider.ExecutionProvider{Record: providercore.Record{Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Extra: map[string]any{"openai_compact_mode": providercore.OpenAICompactModeForceOff, "openai_compact_supported": true}}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gatewayprovider.AllowsCompatibleCompact(tt.provider); got != tt.want {
				t.Fatalf("allowsOpenAICompatibleCompact(...) = %v, want %v", got, tt.want)
			}
		})
	}
}
