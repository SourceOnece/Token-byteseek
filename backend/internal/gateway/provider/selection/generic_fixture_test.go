package selection

import (
	"context"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/gateway/requeststate"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
	gatewaytestkit "github.com/TokenFlux/TokenRouter/internal/gateway/testkit"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/TokenFlux/TokenRouter/internal/usage"
)

// newGenericSelectionForTest 仅组装已有原生能力和配置投影，保持各场景的显式零值。
func newGenericSelectionForTest(deps GenericDependencies, cfg *config.Config) *Generic {
	if deps.Groups == nil {
		deps.Groups = selectionFixtureGroups{}
	}
	if deps.Parameters == nil {
		deps.Parameters = scheduler.NewParameters(scheduler.NewSettingsRuntime(scheduler.Diagnostics{}), nil, diagnosticParameterDefaults(cfg))
	}
	return NewGeneric(deps, selectionOptionsForTest(cfg))
}

func selectionOptionsForTest(cfg *config.Config) Options {
	options := DefaultOptions()
	if cfg == nil {
		return options
	}

	value := cfg.Gateway.Scheduling
	options.Scheduling = scheduler.FlowOptions{LoadBatchEnabled: value.LoadBatchEnabled, PreferSoonestReset: value.PreferSoonestReset, FallbackMaxWaiting: value.FallbackMaxWaiting, StickySessionMaxWaiting: value.StickySessionMaxWaiting, FallbackSelectionMode: value.FallbackSelectionMode, FallbackWaitTimeout: value.FallbackWaitTimeout, StickySessionWaitTimeout: value.StickySessionWaitTimeout}
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

// selectionWindowForTest 复用原用量读取和资金窗口实现，不另建缓存或计算规则。
func selectionWindowForTest(cache billing.WindowCostCache, source usage.UsageLogRepository) *billing.WindowCostGuard {
	return billing.NewWindowCostGuard(cache, gatewaytestkit.WindowCosts(source), billing.WindowCostGuardOptions{Now: time.Now, Stats: billing.SharedWindowCostMetrics(), Log: func(string, ...any) {}, Debug: func(string, ...any) {}})
}

// newGeminiSelectionForTest 将原场景直接接入 Gemini 原生选择器。
func newGeminiSelectionForTest(deps GeminiDependencies, cfg *config.Config) *Gemini {
	if deps.Groups == nil {
		deps.Groups = selectionFixtureGroups{}
	}
	if deps.Parameters == nil {
		deps.Parameters = scheduler.NewParameters(scheduler.NewSettingsRuntime(scheduler.Diagnostics{}), nil, diagnosticParameterDefaults(cfg))
	}
	return NewGemini(deps, selectionOptionsForTest(cfg))
}

// newCompatibleSelectionForTest 只注入原生参数，不初始化供应商执行器。
func newCompatibleSelectionForTest(deps CompatibleDependencies, cfg *config.Config) *Compatible {
	if deps.Groups == nil {
		deps.Groups = selectionFixtureGroups{}
	}
	// 执行夹具惰性提供同一响应归属存储，并通过测试装配注入。
	if deps.Responses == nil {
		cache, _ := deps.Cache.(session.GatewayCache)
		deps.Responses = session.NewOpenAIWSStateStore(cache, gatewayprovider.LogOpenAIWSModeInfo)
	}
	if deps.Parameters == nil {
		deps.Parameters = scheduler.NewParameters(scheduler.NewSettingsRuntime(scheduler.Diagnostics{}), nil, diagnosticParameterDefaults(cfg))
	}
	return NewCompatible(deps, selectionOptionsForTest(cfg))
}

// selectionFixtureGroupID 为只验证调度算法的旧夹具提供明确分组。
func selectionFixtureGroupID(ctx context.Context) *int64 {
	if group, ok := requeststate.GroupFromContext(ctx); ok && group != nil {
		return &group.ID
	}
	if policy, ok := ctx.Value(candidatePolicyKey{}).(candidatePolicy); ok && policy.groupID != nil {
		return policy.groupID
	}
	if input, ok := ctx.Value(selectionRequestKey{}).(selectionRequest); ok && input.groupID != nil {
		return input.groupID
	}
	id := int64(1)
	return &id
}

// prepareSelectionFixtureProvider 显式声明算法夹具的模型范围，专门的模型能力测试直接构造提供商。
func prepareSelectionFixtureProvider(ctx context.Context, value *gatewayprovider.ExecutionProvider, groupID *int64) {
	if value == nil {
		return
	}
	if value.Record.Type == "" {
		value.Record.Type = capability.ProviderTypeAPIKey
	}
	if value.Record.Credentials == nil {
		value.Record.Credentials = map[string]any{}
	}
	if _, set := value.Record.Credentials["model_whitelist"]; !set {
		mapping := provider.ResolveModelMapping(&value.Record, provideradapter.ModelDefaults())
		if len(mapping) == 0 {
			value.Record.Credentials["model_whitelist"] = []string{"*"}
		} else {
			models := make([]string, 0, len(mapping))
			for _, model := range mapping {
				models = append(models, model)
			}
			value.Record.Credentials["model_whitelist"] = models
		}
	}
	if groupID == nil {
		groupID = selectionFixtureGroupID(ctx)
	}
	if len(value.Record.GroupIDs) == 0 && len(value.Record.ProviderGroups) == 0 {
		value.Record.GroupIDs = []int64{*groupID}
	}
}

type selectionFixtureGroups struct{}

func (selectionFixtureGroups) GetByID(_ context.Context, id int64) (*routing.Group, error) {
	return &routing.Group{ID: id, Hydrated: true, Status: routing.StatusActive}, nil
}

func (s selectionFixtureGroups) GetByIDLite(ctx context.Context, id int64) (*routing.Group, error) {
	return s.GetByID(ctx, id)
}
