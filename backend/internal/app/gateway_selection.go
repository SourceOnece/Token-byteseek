package app

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/codexticket"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/selection"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
	"github.com/TokenFlux/TokenRouter/internal/infra/telemetry/logging"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/TokenFlux/TokenRouter/internal/usage"
	usagepostgres "github.com/TokenFlux/TokenRouter/internal/usage/postgres"
)

// provideSelectionReads 只组合原有查询端口；快照与数据库的先后由选择用例保留。
func provideSelectionReads(providers gatewayadapter.ExecutionProviderStore, groups routing.GroupRepository, snapshots selection.Snapshots) selection.Reads {
	return selection.Reads{Providers: providers, Groups: groups, Snapshot: snapshots}
}

// provideSelectionShared 发布共享的反馈、参数、计数、健康和分组策略读取实例。
func provideSelectionShared(cache session.GatewayCache, concurrency *scheduler.ConcurrencyService, health *provideradapter.UpstreamHealth, modelConfigs *routing.PricingConfigService, shared *schedulerSharedState, tickets *codexticket.CodexTicketService) selection.Shared {
	return selection.Shared{
		Tickets:       tickets,
		Cache:         cache,
		Concurrency:   concurrency,
		Health:        health,
		GroupPolicies: modelConfigs,
		Parameters:    shared.Parameters,
		Feedback:      shared.Feedback,
	}
}

// selectionOptions 仅投影选择实际使用的启动配置，不用默认值覆写显式零值。
func selectionOptions(cfg *config.Config) selection.Options {
	options := selection.DefaultOptions()
	value := strings.ToLower(strings.TrimSpace(productEnv("DEBUG_MODEL_ROUTING")))
	options.DebugRouting = value == "1" || value == "true" || value == "yes" || value == "on"
	if cfg == nil {
		return options
	}
	s := cfg.Gateway.Scheduling
	options.Scheduling = scheduler.FlowOptions{LoadBatchEnabled: s.LoadBatchEnabled, PreferSoonestReset: s.PreferSoonestReset, FallbackMaxWaiting: s.FallbackMaxWaiting, StickySessionMaxWaiting: s.StickySessionMaxWaiting, FallbackSelectionMode: s.FallbackSelectionMode, FallbackWaitTimeout: s.FallbackWaitTimeout, StickySessionWaitTimeout: s.StickySessionWaitTimeout}
	ws := cfg.Gateway.OpenAIWS
	options.WS = &egress.OpenAIWSOptions{Enabled: ws.Enabled, ForceHTTP: ws.ForceHTTP, OAuthEnabled: ws.OAuthEnabled, APIKeyEnabled: ws.APIKeyEnabled, ModeRouterV2Enabled: ws.ModeRouterV2Enabled, ResponsesWebsockets: ws.ResponsesWebsockets, ResponsesWebsocketsV2: ws.ResponsesWebsocketsV2}
	options.WSIngressMode = ws.IngressModeDefault
	options.ReadLegacySticky = ws.SessionHashReadOldFallback
	options.WriteLegacySticky = ws.SessionHashDualWriteOld
	if ws.StickySessionTTLSeconds > 0 {
		options.StickyTTL = time.Duration(ws.StickySessionTTLSeconds) * time.Second
	}
	if ws.StickyResponseIDTTLSeconds > 0 {
		options.ResponseTTL = time.Duration(ws.StickyResponseIDTTLSeconds) * time.Second
	}
	return options
}

// provideSelectionModelTransient 由执行反馈与选号共同持有一个模型瞬态状态。
func provideSelectionModelTransient() *provider.ModelTransientState {
	return provider.NewModelTransientState(0)
}

// provideSelectionProxyCircuit 保留原默认值和正数覆盖，执行观测与选号共用同一隔离状态。
func provideSelectionProxyCircuit(cfg *config.Config) *egress.ProxyStreamCircuit {
	options := egress.DefaultProxyStreamCircuitSettings()
	if cfg != nil {
		value := cfg.Gateway.OpenAIProxyStreamCircuit
		options.Disabled = value.Disabled
		if value.FailureThreshold > 0 {
			options.FailureThreshold = value.FailureThreshold
		}
		if value.WindowSeconds > 0 {
			options.FailureWindow = time.Duration(value.WindowSeconds) * time.Second
		}
		if value.TTLSeconds > 0 {
			options.QuarantineTTL = time.Duration(value.TTLSeconds) * time.Second
		}
	}
	return egress.NewProxyStreamCircuit(options)
}

func provideGenericSelection(reads selection.Reads, shared selection.Shared, cfg *config.Config, windows billing.WindowCostCache, source usage.UsageLogRepository, nativeUsage *usagepostgres.Store, rpm scheduler.RPMCache, sessions scheduler.SessionLimitCache, gates *selectionFreeQuotaGates, providers gatewayadapter.ExecutionProviderStore) *selection.Generic {
	stats := usageWindowStats{nativeUsage}
	guard := billing.NewWindowCostGuard(windows, stats, billing.WindowCostGuardOptions{Now: time.Now, Stats: billing.SharedWindowCostMetrics(), Log: func(format string, args ...any) { logging.LegacyPrintf("service.gateway", format, args...) }, Debug: slog.Debug})
	var write func(context.Context, int64, string) error
	if providers != nil {
		write = providers.SetError
	}
	return selection.NewGeneric(selection.GenericDependencies{
		Reads:                   reads,
		Shared:                  shared,
		Window:                  guard,
		WindowPrefetchAvailable: windows != nil && source != nil,
		RPM:                     rpm,
		Sessions:                sessions,
		FreeQuota:               gates.Generic,
		SetProviderError:        write,
	}, selectionOptions(cfg))
}

func provideCompatibleSelection(generic *selection.Generic, gemini *selection.Gemini, reads selection.Reads, shared selection.Shared, cfg *config.Config, responses session.OpenAIWSStateStore, quota *provider.QuotaSettingsCache, blocks *provider.RuntimeBlockState, transient *provider.ModelTransientState, proxy *egress.ProxyStreamCircuit, state *schedulerSharedState, gates *selectionFreeQuotaGates) *selection.Compatible {
	return selection.NewCompatible(selection.CompatibleDependencies{
		Generic: generic, Gemini: gemini,
		Reads:                reads,
		Shared:               shared,
		Responses:            responses,
		QuotaSettings:        quota,
		RuntimeBlocks:        blocks,
		ModelTransient:       transient,
		ProxyCircuit:         proxy,
		StickyStats:          state.Sticky,
		FreeQuota:            gates.Compatible,
		NewAdvancedFreeQuota: gates.Advanced,
	}, selectionOptions(cfg))
}

func provideGeminiSelection(reads selection.Reads, shared selection.Shared, cfg *config.Config, quota *provider.GeminiPrecheck) *selection.Gemini {
	return selection.NewGemini(selection.GeminiDependencies{Reads: reads, Shared: shared, QuotaPrecheck: quota}, selectionOptions(cfg))
}

// productEnv 优先读取新品牌变量；空值沿用未配置语义，显式零值不会回退。
func productEnv(suffix string) string {
	value := os.Getenv("TOKENROUTER_" + suffix)
	if value != "" {
		return value
	}
	return os.Getenv("SUB2API_" + suffix)
}
