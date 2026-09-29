package app

import (
	"context"
	"log"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/pkg/timezone"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/anthropic"
	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
	usagepostgres "github.com/TokenFlux/TokenRouter/internal/usage/postgres"
)

// provideOAuthUsageStats 直接组合原存储批量查询与同一个展示缓存。
func provideOAuthUsageStats(store *usagepostgres.Store, cache *provider.OAuthUsageCache, calendar timezone.Calendar) *provider.LocalUsageStatistics {
	return provider.NewLocalUsageStatistics(newProviderLocalUsageStats(store), cache, provider.LocalUsageStatisticsOptions{Now: time.Now, Today: calendar.Today, Log: log.Printf})
}

// provideOAuthUsageCore 直接绑定原生读取、平台查询和生命周期，不通过旧服务取回实例。
func provideOAuthUsageCore(store *providerpostgres.ProviderStore, usageStore *usagepostgres.Store, cache *provider.OAuthUsageCache, stats *provider.LocalUsageStatistics, gemini *provider.GeminiQuotaService, antigravity *provider.AntigravityQuota, grokView *provider.GrokQuotaView, grok *provider.GrokQuotaService, openAI *provider.OpenAIQuotaService, fetcher provideradapter.ClaudeUsageClient, fingerprints anthropic.FingerprintCache, profiles *egressprovider.TLSProfiles, transport httpclient.UpstreamTransport, settings *provider.QuotaSettingsCache, connections *gatewayhttp.OpenAIWSConnections, manager *lifecycle.Manager, coordinator *provider.OpenAITaskCoordinator) *provider.OAuthUsageService {
	taskOptions := provider.OpenAITaskOptions{
		Read: store.GetByID,
		Register: func(ctx context.Context, value *provider.Record) (string, error) {
			return provideradapter.RegisterAgentIdentityTask(ctx, value, "https://auth.openai.com/api/accounts")
		},
		Persist: func(ctx context.Context, value *provider.Record, credentials map[string]any) error {
			_, err := provider.PersistCredentials(ctx, store, value, credentials, slog.Warn)
			return err
		},
		Invalidate: connections.InvalidateProvider,
	}
	requests := &provideradapter.OAuthUsageTransport{
		Transport: transport, Profiles: profiles, Fingerprints: fingerprints,
		Tasks: coordinator, TaskOptions: taskOptions,
	}
	// 用量查询的会话保持原独立作用域，不与请求执行的会话缓存合并。
	sessions := provideradapter.NewQoderTokenProvider(qoder.SessionBuilder{})
	sessions.SetHTTPUpstream(transport, profiles)
	qoderQuery := &provideradapter.QoderUsage{Sessions: sessions, Transport: transport, Profiles: profiles}
	qoderOptions := qoderQuery.Options()
	qoderOptions.Enrich = provideradapter.EnrichUsageWithProviderError
	options := provider.OAuthUsageOptions{
		Now: time.Now, Jitter: rand.Int64N, Log: log.Printf, Warn: slog.Warn,
		Qoder: qoderOptions,
		Anthropic: func(ctx context.Context, value *provider.Record) (*provider.ClaudeUsageResponse, error) {
			return requests.FetchAnthropic(ctx, value, fetcher)
		},
		OpenAI: provider.OpenAIUsageOptions{
			Probe: requests.ProbeOpenAI,
			Shadow: func(ctx context.Context, id int64, now time.Time) (map[string]any, error) {
				value, err := openAI.QueryUsage(ctx, id)
				if err != nil {
					return nil, err
				}
				return provider.BuildCodexSparkWindowExtraUpdates(value, now), nil
			},
		},
		OpenAIQuotaPause: func(ctx context.Context, value *provider.Record, info *provider.UsageInfo) {
			if value != nil && info != nil && value.IsOpenAI() {
				info.QuotaAutoPaused, _ = provider.EvaluateQuotaAutoPause(value.Platform, value.Extra, settings.GetOpenAIQuotaAutoPauseSettings(ctx), time.Now())
			}
		},
		Antigravity: provider.AntigravityUsageOptions{
			CanFetch: antigravity.CanFetch,
			Fetch: func(ctx context.Context, value *provider.Record) (*provider.UsageInfo, error) {
				result, err := antigravity.FetchQuota(ctx, value, antigravity.GetProxyURL(ctx, value))
				if result == nil {
					return nil, err
				}
				return result.UsageInfo, err
			},
			Degrade: provideradapter.AntigravityDegradedUsage, Enrich: provideradapter.EnrichUsageWithProviderError,
		},
		Gemini: provider.GeminiUsageOptions{
			Location: geminiQuotaLocation, Quota: gemini.QuotaForProvider,
			Totals: func(ctx context.Context, id int64, start, end time.Time) (provider.GeminiUsageTotals, error) {
				rows, err := usageStore.GetModelStatsWithFilters(ctx, start, end, 0, 0, id, 0, nil, nil, nil)
				if err != nil {
					return provider.GeminiUsageTotals{}, err
				}
				values := projectGeminiModelUsage(rows)
				return provider.AggregateGeminiUsage(values), nil
			},
		},
		Grok: provider.GrokUsageOptions{
			Available: func() bool { return true }, StatsAvailable: func() bool { return true },
			Build: grokView.BuildUsageInfo, Enrich: provideradapter.EnrichUsageWithProviderError,
			Probe: func(ctx context.Context, id int64) (*provider.GrokUsageProbe, error) {
				value, err := grok.ProbeBilling(ctx, id)
				if value == nil {
					return nil, err
				}
				return &provider.GrokUsageProbe{Billing: value.Billing, LocalUsage24h: value.LocalUsage24h, LocalUsage7d: value.LocalUsage7d, LocalUsageMonthly: value.LocalUsageMonthly}, err
			},
		},
	}
	core := provider.NewOAuthUsageService(store, cache, stats, options)
	manager.Register(lifecycle.Hook{Name: "ProviderOAuthUsage", StopOrder: 25, Stop: core.StopContext})
	return core
}

// geminiQuotaLocation 保留原洛杉矶日界及加载失败后的固定时区。
func geminiQuotaLocation() *time.Location {
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return time.FixedZone("PST", -8*3600)
	}
	return location
}
