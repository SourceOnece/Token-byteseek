package app

import (
	"context"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"

	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/selection"
	schedulerhttp "github.com/TokenFlux/TokenRouter/internal/scheduler/httpapi"
)

// provideProviderDiagnostics 直接组合只读诊断与真实选择共享的资格、参数和反馈。
func provideProviderDiagnostics(admin *provider.Admin, groups *routing.GroupAdmin, concurrency *scheduler.ConcurrencyService,

	gateway *selection.Generic, openai *selection.Compatible, shared *schedulerSharedState,
) *selection.Diagnostics {
	return selection.NewDiagnostics(providerDiagnosticSource{providers: admin, groups: groups}, selection.Shared{Concurrency: concurrency, Parameters: shared.Parameters, Feedback: shared.Feedback}, gateway, openai)
}

// provideSchedulerDiagnosticsHTTP 直接将只读诊断用例装配到 scheduler HTTP。
func provideSchedulerDiagnosticsHTTP(core *selection.Diagnostics) *schedulerhttp.DiagnosticsHandler {
	return schedulerhttp.NewDiagnosticsHandler(core)
}

// providerDiagnosticSource 投影管理读取结果，调度诊断复用资格检查和反馈实例。
// 此旧提供商形状随网关诊断端口清理一起删除，不持有缓存、锁或评分规则。
type providerDiagnosticSource struct {
	providers *provider.Admin
	groups    *routing.GroupAdmin
}

func (s providerDiagnosticSource) GetProvider(ctx context.Context, id int64) (*gatewayprovider.ExecutionProvider, error) {
	value, err := s.providers.GetProvider(ctx, id)
	return gatewayprovider.NewExecutionProvider(value), err
}

func (s providerDiagnosticSource) GetGroup(ctx context.Context, id int64) (*routing.Group, error) {
	return s.groups.GetGroup(ctx, id)
}

func (s providerDiagnosticSource) ListProvidersForSchedulerScoreFilter(ctx context.Context, platform, kind, status, search string, gid int64, privacy string) ([]gatewayprovider.ExecutionProvider, error) {
	values, err := s.providers.ListProvidersForSchedulerScoreFilter(ctx, platform, kind, status, search, gid, privacy)
	return gatewayprovider.ExecutionProviders(values), err
}

func (s providerDiagnosticSource) ListSchedulableProvidersForAdvancedSchedulerScore(ctx context.Context, gid *int64, platform string) ([]gatewayprovider.ExecutionProvider, error) {
	values, err := s.providers.ListSchedulableProvidersForAdvancedSchedulerScore(ctx, gid, platform)
	return gatewayprovider.ExecutionProviders(values), err
}
