package app

import (
	"context"
	"log"
	"log/slog"
	"time"

	usageerrors "github.com/TokenFlux/TokenRouter/internal/usage"

	"github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/config"

	"github.com/TokenFlux/TokenRouter/internal/settings"
)

// provideGeminiQuotaPolicy 分开静态配置投影与动态 settings 读取，不构造第二份策略缓存。
func provideGeminiQuotaPolicy(cfg *config.Config, store *settings.Store) *provider.GeminiQuotaService {
	tiers := make(map[string]provider.GeminiTierQuotaOverride, len(cfg.Gemini.Quota.Tiers))
	for id, v := range cfg.Gemini.Quota.Tiers {
		tiers[id] = provider.GeminiTierQuotaOverride{ProRPD: v.ProRPD, FlashRPD: v.FlashRPD, CooldownMinutes: v.CooldownMinutes}
	}
	return provider.NewGeminiQuotaService(provider.GeminiQuotaOptions{StaticTiers: tiers, StaticPolicy: cfg.Gemini.Quota.Policy, Now: time.Now, Log: log.Printf, NotFound: settings.ErrSettingNotFound, LoadPolicy: func(ctx context.Context) (string, error) {
		return store.GetValue(ctx, provider.GeminiQuotaPolicySettingKey)
	}})
}

// provideGeminiPrecheck 保留洛杉矶日界与独立日统计缓存，不持有全量配置。
func provideGeminiPrecheck(policy *provider.GeminiQuotaService, usage usageerrors.UsageLogRepository) *provider.GeminiPrecheck {
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		location = time.FixedZone("PST", -8*3600)
	}
	return provider.NewGeminiPrecheck(policy, newProviderGeminiUsageReader(usage), provider.GeminiPrecheckOptions{Now: time.Now, Location: location, Info: slog.Info})
}
