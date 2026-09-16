package service

import (
	"context"

	"github.com/tidwall/gjson"
)

type openCodeGoUsageAdapter struct{}

func (*openCodeGoUsageAdapter) Name() string { return UpstreamUsageAdapterOpenCodeGo }

// Go 使用同一配置主机和路径前缀；Zen 不具备本批采用的订阅窗口接口。
func (*openCodeGoUsageAdapter) Query(ctx context.Context, client *upstreamUsageHTTPClient) (*UpstreamUsageInfo, error) {
	if !client.account.IsOpenCodeGoPlan() {
		return nil, ErrUpstreamUsageUnsupported
	}
	endpoint := openCodeGoQuotaURL(client.baseURL)
	if endpoint == "" {
		return nil, ErrUpstreamUsageConfigInvalid
	}
	body, status, err := client.getURL(ctx, endpoint, true)
	if err != nil {
		return nil, err
	}
	if err := validateCNUsageStatus(status); err != nil {
		return nil, err
	}
	return cnUsageLimits(PlatformOpenCodeGo, parseOpenCodeGoUsageTiers(body))
}

func parseOpenCodeGoUsageTiers(body []byte) []CNQuotaTier {
	usage := gjson.GetBytes(body, "usage")
	if !usage.IsObject() {
		return nil
	}
	var tiers []CNQuotaTier
	for _, field := range []struct{ key, window string }{{"rolling", "5h"}, {"weekly", "weekly"}, {"monthly", "monthly"}} {
		node := usage.Get(field.key)
		used, ok := cnParseF64(node.Get("percent").Value())
		if !ok || !validFiniteNumber(used) || used < 0 {
			continue
		}
		tiers = append(tiers, CNQuotaTier{Window: field.window, UsedPercent: used, ResetAt: cnNormalizeResetTime(node.Get("resetsAt").Value())})
	}
	return tiers
}
