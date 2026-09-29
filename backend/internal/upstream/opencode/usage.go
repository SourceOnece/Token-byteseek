package opencode

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/upstream/internal/usageclient"
	"github.com/TokenFlux/TokenRouter/internal/upstream/usagecontract"
	"github.com/TokenFlux/TokenRouter/internal/upstream/usageview"
	"github.com/tidwall/gjson"
)

type GoUsageAdapter struct{}

func (*GoUsageAdapter) Name() string { return usageview.UpstreamUsageAdapterOpenCodeGo }

// Go 使用同一配置主机和路径前缀；Zen 不具备本批采用的订阅窗口接口。
func (*GoUsageAdapter) Query(ctx context.Context, input *usagecontract.Request) (*usageview.UpstreamUsageInfo, error) {
	client := usageclient.New(input)
	if !input.OpenCodeGo {
		return nil, usageview.ErrUpstreamUsageUnsupported
	}
	endpoint, buildErr := usageclient.UpstreamUsageEndpoint(client.BaseURL, "/v1/usage", client.Endpoint)
	if buildErr != nil {
		return nil, usageview.ErrUpstreamUsageConfigInvalid
	}
	body, status, err := client.GetURL(ctx, endpoint, true)
	if err != nil {
		return nil, err
	}
	if err := usageclient.ValidateCNUsageStatus(status); err != nil {
		return nil, err
	}
	return usageclient.CnUsageLimits("opencode_go", parseOpenCodeGoUsageTiers(body))
}

func parseOpenCodeGoUsageTiers(body []byte) []usageview.CNQuotaTier {
	usage := gjson.GetBytes(body, "usage")
	if !usage.IsObject() {
		return nil
	}
	var tiers []usageview.CNQuotaTier
	for _, field := range []struct{ key, window string }{{"rolling", "5h"}, {"weekly", "weekly"}, {"monthly", "monthly"}} {
		node := usage.Get(field.key)
		used, ok := usageclient.CnParseF64(node.Get("percent").Value())
		if !ok || !usageview.ValidFiniteNumber(used) || used < 0 {
			continue
		}
		tiers = append(tiers, usageview.CNQuotaTier{Window: field.window, UsedPercent: used, ResetAt: usageclient.CnNormalizeResetTime(node.Get("resetsAt").Value())})
	}
	return tiers
}
