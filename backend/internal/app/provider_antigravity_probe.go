package app

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/pkg/logredact"

	"github.com/TokenFlux/TokenRouter/internal/upstream/gemini"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/infra/telemetry/logging"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/TokenFlux/TokenRouter/internal/scheduler/rediscache/codec"
	"github.com/TokenFlux/TokenRouter/internal/upstream/antigravity"
)

// provideAntigravityRetry 在健康配置绑定完成后组合唯一平台适配，尚存网关只持有同一实例。
func provideAntigravityRetry(store *providerpostgres.ProviderStore, counter provider.Internal500CounterCache, runtime *providerHealthRuntime, snapshots *scheduler.SnapshotService, transport httpclient.UpstreamTransport, cfg *config.Config) *provideradapter.AntigravityRetry {
	health := &provider.AntigravityHealth{
		Store:     store,
		Counter:   counter,
		ModelKeys: provideradapter.AntigravityModelLimitKeys,
		Error:     slog.Error,
		Warn:      slog.Warn,
		Info:      slog.Info,

		Logf: func(format string, values ...any) {
			logging.LegacyPrintf("service.antigravity_gateway", format, values...)
		},
	}
	if snapshots != nil {
		health.Publish = func(ctx context.Context, value *provider.Record) error {
			return snapshots.UpdateProviderInCache(ctx, codec.WrapRecord(value))
		}
	}
	core := &provideradapter.AntigravityRetry{
		Health: health,
		Policy: runtime.Health,
		Do:     transport.Do,

		BaseURL: func(value *provider.Record) string {
			return antigravity.ResolveAntigravityForwardBaseURL(os.Getenv("GATEWAY_ANTIGRAVITY_FORWARD_BASE_URL"), provideradapter.AntigravityPaidTier(value))
		},

		BodyLimit: func() int64 {
			limit := int64(512 << 10)
			if cfg.Gateway.LogUpstreamErrorBody && cfg.Gateway.LogUpstreamErrorBodyMaxBytes > int(limit) {
				return int64(cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
			}
			return limit
		},

		LogConfig: func() (bool, int) { return cfg.Gateway.LogUpstreamErrorBody, cfg.Gateway.LogUpstreamErrorBodyMaxBytes },

		TruncateString: logredact.TruncateUTF8,
		SafeURL:        logredact.SafeUpstreamURL,
	}
	return core
}

// provideAntigravityProbe 与转发共用重试、提供商健康和尝试拥有者；不登记第二个后台实例。
func provideAntigravityProbe(tokens *provider.AntigravityTokenSource, retry *provideradapter.AntigravityRetry, activity *gatewayRequestActivity) *provideradapter.AntigravityProbe {
	return &provideradapter.AntigravityProbe{Tokens: tokens, Retry: retry, Enter: activity.Enter}
}

// provideAntigravityErrorObserver 复用重试器的健康拥有者，观测只按既有顺序写入提供商状态。
func provideAntigravityErrorObserver(core *provideradapter.AntigravityRetry, store *providerpostgres.ProviderStore, runtime *providerHealthRuntime, cfg *config.Config) *provideradapter.AntigravityErrorObserver {
	health := core.Health

	return &provideradapter.AntigravityErrorObserver{
		Health: health,

		LogConfig: func() (bool, int) {
			limit := 2048
			if cfg.Gateway.LogUpstreamErrorBodyMaxBytes > 0 {
				limit = cfg.Gateway.LogUpstreamErrorBodyMaxBytes
			}
			return cfg.Gateway.LogUpstreamErrorBody, limit
		},

		TruncateString: logredact.TruncateUTF8,

		ResetTime: func(body []byte) *int64 {
			return gemini.ParseGeminiRateLimitResetTime(body, func() *int64 { v := provider.GeminiDailyResetTime(time.Now(), geminiQuotaLocation()).Unix(); return &v })
		},

		DefaultDuration: func() time.Duration {
			return provideradapter.AntigravityFallbackDuration(cfg.Gateway.AntigravityFallbackCooldownMinutes, os.Getenv("GATEWAY_ANTIGRAVITY_FALLBACK_COOLDOWN_SECONDS"))
		},

		SetRateLimited: store.SetRateLimited,
		Other:          runtime.Observer,
	}
}
