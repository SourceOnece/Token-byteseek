package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// grokImportQuotaProbe 只把旧额度服务的观测投影给提供商导入探测器。
type grokImportQuotaProbe struct{ source *provider.GrokQuotaService }

func (p grokImportQuotaProbe) QueryQuota(ctx context.Context, id int64) (*provider.GrokImportProbeResult, error) {
	value, err := p.source.QueryQuota(ctx, id)
	if value == nil {
		return nil, err
	}
	return &provider.GrokImportProbeResult{Model: value.Model, StatusCode: value.StatusCode, HeadersObserved: value.HeadersObserved}, err
}

// provideProviderArchive 显式组合文件用例，代理、提供商、探测和隐私使用唯一生产实例。
func provideProviderArchive(admin *provider.Admin, proxies *egress.ProxyTransfer, privacy *provider.PrivacyService, settings *provider.RuntimeSettings, probes *provider.GrokImportProbeScheduler, grok *provider.GrokQuotaService, tasks *lifecycle.Tasks) *provider.Archive {
	options := provider.ArchiveOptions{
		Now: time.Now, Info: slog.Info, Error: slog.Error, Debug: slog.Debug, DecodeIDToken: provideradapter.DecodeArchiveIDToken, Background: tasks.Go, ForcePrivacy: privacy.ForceAntigravityPrivacy, Defaults: settings.GetOpenAIOAuthImportDefaults,
		Probe: func(snapshot provider.ProviderSnapshot) {
			probes.Schedule(grokImportQuotaProbe{source: grok}, &snapshot)
		},
	}
	return provider.NewArchive(admin, proxies, options)
}
