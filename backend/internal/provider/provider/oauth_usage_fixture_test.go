package provider

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"time"

	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
)

// 夹具仅组合真实用量核心与平台端口，不复制缓存、回源、恢复或写入算法。
type oauthUsageFixtureOptions struct {
	providerRepo         provider.OAuthUsageReader
	cache                *provider.OAuthUsageCache
	httpUpstream         QoderTransport
	tlsFPProfileService  *egressprovider.TLSProfiles
	qoderSessionProvider *QoderTokenProvider
}

func newOAuthUsageFixture(options oauthUsageFixtureOptions) *provider.OAuthUsageService {
	sessions := options.qoderSessionProvider
	if sessions == nil {
		sessions = NewQoderTokenProvider(qoder.SessionBuilder{})
		sessions.SetHTTPUpstream(options.httpUpstream, options.tlsFPProfileService)
	}
	qoderQuery := &QoderUsage{Sessions: sessions, Transport: options.httpUpstream, Profiles: options.tlsFPProfileService}
	qoderOptions := qoderQuery.Options()
	qoderOptions.Enrich = EnrichUsageWithProviderError
	requests := &OAuthUsageTransport{Transport: options.httpUpstream, Profiles: options.tlsFPProfileService}
	return provider.NewOAuthUsageService(options.providerRepo, options.cache, nil, provider.OAuthUsageOptions{
		Now: time.Now, Log: log.Printf, Warn: slog.Warn,
		Qoder:  qoderOptions,
		OpenAI: provider.OpenAIUsageOptions{Probe: requests.ProbeOpenAI},
		OpenAIQuotaPause: func(_ context.Context, value *provider.Record, info *provider.UsageInfo) {
			if value != nil && info != nil {
				info.QuotaAutoPaused, _ = provider.EvaluateQuotaAutoPause(value.Platform, value.Extra, provider.QuotaAutoPauseSettings{}, time.Now())
			}
		},
	})
}

// 测试存储按 ID 返回独立记录，缺失时返回对应错误。
type usageRecordFixture struct{ providers []provider.Record }

func (r usageRecordFixture) GetByID(_ context.Context, id int64) (*provider.Record, error) {
	for index := range r.providers {
		if r.providers[index].ID == id {
			return provider.CloneRecord(&r.providers[index]), nil
		}
	}
	return nil, errors.New("provider not found")
}

func (r usageRecordFixture) GetByIDs(_ context.Context, ids []int64) ([]*provider.Record, error) {
	result := make([]*provider.Record, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		for index := range r.providers {
			if r.providers[index].ID == id {
				result = append(result, provider.CloneRecord(&r.providers[index]))
				break
			}
		}
	}
	return result, nil
}

// 原断言观察这三个写入端口，真实条件写与事务另由 PostgreSQL 契约覆盖。
func (r *providerUsageCodexProbeRepo) UpdateUsageExtraIfUnchanged(ctx context.Context, version provider.UsageObservationVersion, updates map[string]any) (bool, error) {
	return true, r.UpdateExtra(ctx, version.ID, updates)
}

func (r *providerUsageCodexProbeRepo) SetUsageRateLimitIfUnchanged(ctx context.Context, version provider.UsageObservationVersion, reset time.Time) (bool, error) {
	return true, r.SetRateLimited(ctx, version.ID, reset)
}

func (r *providerUsageCodexProbeRepo) ClearUsageRateLimitIfUnchanged(ctx context.Context, version provider.UsageObservationVersion) (bool, error) {
	return true, r.ClearRateLimit(ctx, version.ID)
}
