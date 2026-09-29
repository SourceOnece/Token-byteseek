package selection

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	schedulercore "github.com/TokenFlux/TokenRouter/internal/scheduler"
)

// Generic 只组合通用选择所需的读取、资格和资源；尝试切换与平台转发不属于本对象。
type Generic struct {
	tickets interface {
		Blocks(context.Context, *provider.Record, string) bool
		VerifiedFlow(int64) bool
	}
	options                 Options
	providerRepo            Providers
	groupRepo               Groups
	schedulerSnapshot       Snapshots
	cache                   schedulercore.StickyCache
	concurrencyService      *schedulercore.ConcurrencyService
	healthObserver          *provideradapter.UpstreamHealth
	groupPolicies           *routing.PricingConfigService
	schedulerParameters     *schedulercore.Parameters
	advancedProviderStats   *schedulercore.RuntimeStats
	freeQuotaGate           *provider.FreeQuotaGate
	window                  *billing.WindowCostGuard
	windowPrefetchAvailable bool
	rpmCache                schedulercore.RPMCache
	sessionLimitCache       schedulercore.SessionLimitCache
	setProviderError        func(context.Context, int64, string) error
}

// NewGeneric 绑定唯一状态拥有者，执行凭据只留在单次选择的适配作用域。
// @project-doc docs/architecture/provider_scheduling_and_cache.md#advanced_scheduler_selection
func NewGeneric(deps GenericDependencies, options Options) *Generic {
	if deps.Window == nil {
		deps.Window = defaultWindowCostGuard()
	}
	return &Generic{
		tickets:           deps.Tickets,
		options:           options,
		providerRepo:      deps.Providers,
		groupRepo:         deps.Groups,
		schedulerSnapshot: deps.Snapshot,
		cache:             deps.Cache,

		concurrencyService:  deps.Concurrency,
		healthObserver:      deps.Health,
		groupPolicies:       deps.GroupPolicies,
		schedulerParameters: deps.Parameters,

		advancedProviderStats:   deps.Feedback,
		freeQuotaGate:           deps.FreeQuota,
		window:                  deps.Window,
		windowPrefetchAvailable: deps.WindowPrefetchAvailable,

		rpmCache:          deps.RPM,
		sessionLimitCache: deps.Sessions,
		setProviderError:  deps.SetProviderError,
	}
}

// Compatible 在同一原生调度器上连接 OpenAI/Grok 资格，借用共享状态而不拥有供应商执行。
type Compatible struct {
	tickets interface {
		Blocks(context.Context, *provider.Record, string) bool
		VerifiedFlow(int64) bool
	}
	generic             *Generic
	gemini              *Gemini
	options             Options
	providerRepo        Providers
	schedulerSnapshot   Snapshots
	schedulingGroups    func(context.Context, int64) (*routing.Group, error)
	cache               schedulercore.StickyCache
	concurrencyService  *schedulercore.ConcurrencyService
	healthObserver      *provideradapter.UpstreamHealth
	groupPolicies       *routing.PricingConfigService
	schedulerParameters *schedulercore.Parameters
	openaiProviderStats *schedulercore.RuntimeStats
	quotaSettings       *provider.QuotaSettingsCache
	freeQuotaGate       *provider.FreeQuotaGate
	newFreeQuotaGate    func() *provider.FreeQuotaGate
	runtime             *provider.RuntimeBlockState
	modelTransient      *provider.ModelTransientState
	proxyCircuit        *egress.ProxyStreamCircuit
	proxyFailOpenLogAt  atomic.Int64
	responseState       session.OpenAIWSStateStore
	stickyMetrics       *schedulercore.StickyStats
	pickerOnce          sync.Once
	picker              pickerEngine
}

func NewCompatible(deps CompatibleDependencies, options Options) *Compatible {
	if deps.RuntimeBlocks == nil {
		deps.RuntimeBlocks = provider.NewRuntimeBlockState(time.Now)
	}
	if deps.ModelTransient == nil {
		deps.ModelTransient = provider.NewModelTransientState(0)
	}
	if deps.ProxyCircuit == nil {
		deps.ProxyCircuit = egress.NewProxyStreamCircuit(egress.DefaultProxyStreamCircuitSettings())
	}
	if deps.StickyStats == nil {
		deps.StickyStats = &schedulercore.StickyStats{}
	}
	var groups func(context.Context, int64) (*routing.Group, error)
	if deps.Groups != nil {
		groups = deps.Groups.GetByID
	}
	return &Compatible{
		tickets: deps.Tickets,
		generic: deps.Generic, gemini: deps.Gemini,
		options:           options,
		providerRepo:      deps.Providers,
		schedulerSnapshot: deps.Snapshot,
		schedulingGroups:  groups,
		cache:             deps.Cache,

		concurrencyService:  deps.Concurrency,
		healthObserver:      deps.Health,
		groupPolicies:       deps.GroupPolicies,
		schedulerParameters: deps.Parameters,

		openaiProviderStats: deps.Feedback,
		quotaSettings:       deps.QuotaSettings,
		freeQuotaGate:       deps.FreeQuota,
		newFreeQuotaGate:    deps.NewAdvancedFreeQuota,

		runtime:        deps.RuntimeBlocks,
		modelTransient: deps.ModelTransient,
		proxyCircuit:   deps.ProxyCircuit,
		responseState:  deps.Responses,
		stickyMetrics:  deps.StickyStats,
	}
}

// Gemini 仅持有 Gemini/混合池的无槽选择依赖，凭据和报文执行留在各自拥有者。
type Gemini struct {
	options               Options
	providerRepo          Providers
	groupRepo             Groups
	schedulerSnapshot     Snapshots
	cache                 schedulercore.StickyCache
	schedulerParameters   *schedulercore.Parameters
	advancedProviderStats *schedulercore.RuntimeStats
	quotaPrecheck         *provider.GeminiPrecheck
}

func NewGemini(deps GeminiDependencies, options Options) *Gemini {
	return &Gemini{
		options:           options,
		providerRepo:      deps.Providers,
		groupRepo:         deps.Groups,
		schedulerSnapshot: deps.Snapshot,
		cache:             deps.Cache,

		schedulerParameters:   deps.Parameters,
		advancedProviderStats: deps.Feedback,
		quotaPrecheck:         deps.QuotaPrecheck,
	}
}

// DiagnosticSource 只允许原诊断读取；安全 DTO 仍由 scheduler 核心产生。
type DiagnosticSource interface {
	GetProvider(context.Context, int64) (*gatewayadapter.ExecutionProvider, error)
	GetGroup(context.Context, int64) (*routing.Group, error)
	ListProvidersForSchedulerScoreFilter(context.Context, string, string, string, string, int64, string) ([]gatewayadapter.ExecutionProvider, error)
	ListSchedulableProvidersForAdvancedSchedulerScore(context.Context, *int64, string) ([]gatewayadapter.ExecutionProvider, error)
}

// Diagnostics 与真实选择共用参数、反馈和资格实例，不抢槽或写入粘性。
type Diagnostics struct {
	source              DiagnosticSource
	concurrencyService  *schedulercore.ConcurrencyService
	feedback            *schedulercore.RuntimeStats
	schedulerParameters *schedulercore.Parameters
	gatewayService      *Generic
	openAIGateway       *Compatible
}

func NewDiagnostics(source DiagnosticSource, shared Shared, generic *Generic, compatible *Compatible) *Diagnostics {
	return &Diagnostics{
		source:              source,
		concurrencyService:  shared.Concurrency,
		feedback:            shared.Feedback,
		schedulerParameters: shared.Parameters,

		gatewayService: generic,
		openAIGateway:  compatible,
	}
}
