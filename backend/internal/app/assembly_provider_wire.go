//go:build wireinject

package app

import (
	tickethttp "github.com/TokenFlux/TokenRouter/internal/codexticket/httpapi"
	providerauth "github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	qualityhttp "github.com/TokenFlux/TokenRouter/internal/quality/httpapi"

	providerhttp "github.com/TokenFlux/TokenRouter/internal/provider/httpapi"

	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"

	providerredis "github.com/TokenFlux/TokenRouter/internal/provider/rediscache"

	"github.com/google/wire"
)

// 提供商管理、授权与运行能力的组合根登记；这里只分组原 provider，不创建资源或复制业务实现。
var providerAssemblyProviders = wire.NewSet(
	provideCodexTickets,
	provideCodexQuality,
	qualityhttp.New,
	tickethttp.New,
	provideOpenAITestExecutor,
	providerauth.NewOAuthUsageCache,
	provideGeminiAuthorization,
	provideProviderProbeTasks,
	provideGrokAuthorization,
	provideGrokTokens,
	provideOpenAIAuthorization,
	provideOpenAITokens,
	provideOpenAIExecutionCredentials,
	provideAgentTaskCoordinator,
	provideClaudeTokens,
	provideMessageCredentials,
	provideGeminiTokens,
	provideAntigravityTokens,
	wire.Bind(new(providerauth.GrokRefreshTokenService), new(*providerauth.GrokAuthorization)),
	provideAntigravityAuthorization,
	provideClaudeAuthorization,
	provideTokenCacheInvalidator,
	provideQoderTokens,
	provideQoderRequestRefresh,
	provideQoderAuthorization,
	providerAuthorizationHTTPProviders,
	wire.Bind(new(providerhttp.GeminiAuthorizationUseCase), new(*providerauth.GeminiAuthorization)),
	wire.Bind(new(providerhttp.AntigravityAuthorizationUseCase), new(*providerauth.AntigravityAuthorization)),
	provideOpenAIQuota,
	provideGrokQuota,
	provideCodexInvites,
	wire.Bind(new(providerhttp.CodexInviteResetCommands), new(*providerauth.CodexInviteResetService)),
	provideOpenAIProviderOAuth,
	providerredis.NewTempUnschedCache,
	providerredis.NewTimeoutCounterCache,
	providerredis.NewOpenAI403CounterCache,
	provideGeminiPrecheck,
	provideGeminiQuotaPolicy,
	provideAntigravityQuota,
	provideradapter.NewGrokQuotaView,
	provideProviderModelSync,
	provideProviderTier,
	provideProviderManagementList,
	provideProviderRuntimePresenter,
	provideProviderRuntimeState,
	provideProviderHealthRuntime,
	provideProviderRecovery,
	provideProviderManagement,
	provideManagedRefresh,
	wire.Bind(new(providerauth.RefreshFailureObserver), new(*providerauth.RuntimeBlockState)),
	wire.Bind(new(providerauth.RuntimeUnblocker), new(*providerauth.RuntimeBlockState)),
	provideRefreshPlatforms,
	provideRefreshPostActions,
	provideBackgroundRefresh,
	wire.Bind(new(providerauth.GrokOAuthReconciler), new(*providerauth.BackgroundRefreshService)),
	provideOAuthUsageCore,
	provideOAuthUsageStats,
	providerhttp.NewOAuthUsageHandler,
	provideOllamaUsage,
	providerhttp.NewOllamaUsageHandler,
	provideCodexImporter,
	providerhttp.NewCodexImportHandler,
	provideCRSSync,
	provideCRSHTTP,
	provideProviderArchive,
	providerhttp.NewArchiveHandler,
	provideProviderImportProbes,
	provideGrokOAuthWithImports,
	provideCNUsageMonitor,
	provideProviderTestHTTP,
	provideUpstreamUsageHTTP,
	provideUpstreamUsage,
	provideProviderTests,
	provideAntigravityRetry,
	provideAntigravityProbe,
	provideScheduledTestRunner,
	provideScheduledTests,
	providerpostgres.NewScheduledTestPlanRepository,
	providerpostgres.NewScheduledTestResultRepository,
	providerhttp.NewScheduledTestHandler,
	provideProviderPrivacy,
	provideProviderAdmin,
	provideProviderExpiry,
	provideProviderDeferred,
	provideProviderRefresh,
	provideProviderStore,
	provideOAuthTokenCache,
	provideQuotaSettings,
	provideProviderSettings,
	providerhttp.NewRuntimeSettingsHandler,
)
