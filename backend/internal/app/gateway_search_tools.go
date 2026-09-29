package app

import (
	"context"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/selection"

	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/transport"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/gateway/admission"
	"github.com/TokenFlux/TokenRouter/internal/gateway/completion"
	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/searchtools"
	"github.com/TokenFlux/TokenRouter/internal/moderation"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/TokenFlux/TokenRouter/internal/search"
)

// ProvideGatewaySearchTools 由组合根为请求链持有唯一工具编排器；不创建新 Manager 或配额状态。
func ProvideGatewaySearchTools(settings *search.ConfigService, modelConfigs *routing.PricingConfigService) *searchtools.Emulator {
	runtime := gatewayprovider.NewSearchTools(settings, modelConfigs)
	return runtime
}

// standaloneSearchExecution 只把原有单次选号与执行结果投影给原生搜索入口。
type standaloneSearchExecution struct {
	selector *selection.Compatible
	executor *gatewayprovider.GrokSearchExecutor
}

func (p standaloneSearchExecution) Select(ctx context.Context, group int64, model string, excluded map[int64]struct{}) (gatewayhttp.StandaloneSearchTarget, searchtools.Selection, bool, error) {
	selected, _, err := p.selector.SelectProviderWithSchedulerForCapability(ctx, &group, "", "", model, excluded, egress.OpenAIUpstreamTransportHTTPSSE, provider.OpenAIEndpointCapabilityTextGeneration, false, false, capability.PlatformGrok)
	if err != nil {
		return nil, searchtools.Selection{}, false, err
	}
	if selected == nil || selected.Provider == nil {
		return nil, searchtools.Selection{}, false, nil
	}
	target := standaloneSearchTarget{source: p.executor, provider: gatewayprovider.ExecutionRecord(selected.Provider)}
	return target, searchtools.Selection{ProviderID: selected.Provider.Record.ID, Acquired: selected.Acquired, Release: selected.ReleaseFunc, WaitPlan: selected.WaitPlan}, true, nil
}

// 目标只在该请求内保存已选实例，完成入队时才投影独立快照。
type standaloneSearchTarget struct {
	source   *gatewayprovider.GrokSearchExecutor
	provider *provider.Record
}

func (t standaloneSearchTarget) CompletionRecord() *provider.Record {
	return t.provider
}

func (t standaloneSearchTarget) Execute(ctx context.Context, body []byte) ([]byte, error) {
	return t.source.Execute(ctx, t.provider, body)
}

// ProvideGatewaySearchHTTP 直接构造原生入口，沿用唯一选号、资金、审核和完成运行时。
func ProvideGatewaySearchHTTP(executor *gatewayprovider.GrokSearchExecutor, selector *selection.Compatible, funding *admission.FundingAdmission, concurrency *scheduler.ConcurrencyService, keys *apikey.APIKeyService, moderator *moderation.ContentModerationService, workers *completion.UsageRecordWorkerPool, cfg *config.Config, _ *searchtools.Emulator, activity *gatewayRequestActivity, recorders GatewayCompletionRecorders) *gatewayhttp.SearchHandler {
	interval := time.Duration(0)
	if cfg != nil {
		interval = time.Duration(cfg.Concurrency.PingInterval) * time.Second
	}
	ports := gatewayhttp.SearchPorts{Selector: standaloneSearchExecution{selector, executor}, Funding: funding, Concurrency: gatewayhttp.NewConcurrencyHelper(concurrency, gatewayhttp.SSEPingFormatClaude, interval), Recorder: recorders.Forward, Workers: workers}
	if keys != nil {
		ports.Keys = keys
	}
	if moderator != nil {
		ports.Moderation = moderator
	}
	result := gatewayhttp.NewSearchHandler(ports)
	if activity != nil {
		result.BindRequestActivity(activity.Enter)
	}
	return result
}

// provideGrokSearchExecutor 共享已有传输，动态地址仍在每次请求的原读取时点取得。
func provideGrokSearchExecutor(client *transport.Client, readers *gatewayprovider.RuntimeReaders) *gatewayprovider.GrokSearchExecutor {
	out := &gatewayprovider.GrokSearchExecutor{Transport: client}
	if readers != nil {
		out.DefaultBaseURL = func() string {
			return gatewayprovider.GrokBaseURLForMode(readers.Gateway.GetGrokDefaultBaseURLMode(context.Background()))
		}
	}
	return out
}
