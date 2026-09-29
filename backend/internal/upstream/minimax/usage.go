package minimax

import (
	"context"
	"strings"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/upstream/internal/usageclient"
	"github.com/TokenFlux/TokenRouter/internal/upstream/usagecontract"
	"github.com/TokenFlux/TokenRouter/internal/upstream/usageview"
	"github.com/tidwall/gjson"
)

// MiniMax 复用本地只读适配器和传输边界，不另起直接 HTTP 请求或改变配置主机。
type CodingUsageAdapter struct{}

func (*CodingUsageAdapter) Name() string { return usageview.UpstreamUsageAdapterMiniMaxCoding }
func (*CodingUsageAdapter) Query(ctx context.Context, input *usagecontract.Request) (*usageview.UpstreamUsageInfo, error) {
	client := usageclient.New(input)
	endpoint, err := usageclient.CnUsageEndpoint(client.BaseURL, "/v1/api/openplatform/coding_plan/remains", false)
	if err != nil {
		return nil, usageview.ErrUpstreamUsageConfigInvalid.WithCause(err)
	}
	body, status, err := client.GetURLWithHeaders(ctx, endpoint, map[string]string{
		"Authorization": "Bearer " + client.APIKey, "Content-Type": "application/json", "Accept-Language": "en-US,en",
	})
	if err != nil {
		return nil, err
	}
	if err := usageclient.ValidateCNUsageStatus(status); err != nil {
		return nil, err
	}
	if code := gjson.GetBytes(body, "base_resp.status_code"); code.Exists() && code.Int() != 0 {
		return nil, usageview.ErrUpstreamUsageInvalidResponse
	}
	return usageclient.CnUsageLimits("minimax", parseMiniMaxUsageTiers(body))
}

// 只读取 general 编程套餐，不把视频额度或缺少的字段当作可用额度。
func parseMiniMaxUsageTiers(body []byte) []usageview.CNQuotaTier {
	remains := gjson.GetBytes(body, "model_remains")
	if !remains.IsArray() {
		return nil
	}
	var general gjson.Result
	remains.ForEach(func(_, item gjson.Result) bool {
		if strings.EqualFold(strings.TrimSpace(item.Get("model_name").String()), "general") {
			general = item
			return false
		}
		return true
	})
	if !general.Exists() {
		return nil
	}
	var tiers []usageview.CNQuotaTier
	add := func(window, percentField, resetField string) {
		remaining, ok := usageclient.CnParseF64(general.Get(percentField).Value())
		if !ok || !usageview.ValidFiniteNumber(remaining) || remaining < 0 || remaining > 100 {
			return
		}
		tiers = append(tiers, usageview.CNQuotaTier{Window: window, UsedPercent: 100 - remaining, ResetAt: minimaxResetTime(general.Get(resetField))})
	}
	add("5h", "current_interval_remaining_percent", "end_time")
	if general.Get("current_weekly_status").Int() == 1 {
		add("weekly", "current_weekly_remaining_percent", "weekly_end_time")
	}
	return tiers
}

func minimaxResetTime(value gjson.Result) string {
	if !value.Exists() {
		return ""
	}
	timestamp := value.Int()
	if timestamp <= 0 {
		return usageclient.CnNormalizeResetTime(value.Value())
	}
	if timestamp < 1_000_000_000_000 {
		return time.Unix(timestamp, 0).UTC().Format(time.RFC3339)
	}
	return time.UnixMilli(timestamp).UTC().Format(time.RFC3339)
}
