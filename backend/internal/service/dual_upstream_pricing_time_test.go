//go:build unit

package service

import (
	"context"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// 同一固定请求时刻必须得到相同售价和账号默认成本；历史 Pro 价不受当前时间干扰。
func TestDeepseekPriceAndAccountCostShareRequestTime(t *testing.T) {
	bs := NewBillingService(&config.Config{}, nil)
	tokens := UsageTokens{InputTokens: 1000000, OutputTokens: 1000000, CacheReadTokens: 1000000}
	cases := []struct {
		at   time.Time
		want float64
	}{
		{time.Date(2026, 9, 14, 3, 59, 59, 0, time.UTC), 2 * (.66 + 1.98 + .022)},
		{time.Date(2026, 9, 14, 4, 0, 0, 0, time.UTC), .15 + .60 + .003},
		{time.Date(2026, 9, 14, 6, 0, 0, 0, time.UTC), 2 * (.15 + .60 + .003)},
	}
	for _, tc := range cases {
		t.Run(tc.at.String(), func(t *testing.T) {
			cost, err := bs.CalculateCostUnified(CostInput{Ctx: context.Background(), Model: "deepseek-v4-pro", Tokens: tokens, RateMultiplier: 1, PricingAt: tc.at, Resolver: NewModelPricingResolver(nil, bs)})
			require.NoError(t, err)
			require.InDelta(t, tc.want, cost.TotalCost, 1e-12)
			stats := tryModelFilePricingAt(bs, "deepseek-v4-pro", tokens, "", tc.at)
			require.NotNil(t, stats)
			require.InDelta(t, cost.TotalCost, *stats, 1e-12)
		})
	}
}

func TestGLM53FallbacksRemainDistinct(t *testing.T) {
	bs := NewBillingService(&config.Config{}, nil)
	for _, tc := range []struct {
		model         string
		input, output float64
	}{
		{"glm-5.3", 1.4e-6, 4.4e-6}, {"glm-5.3-flash", .15e-6, .5e-6}, {"glm-5.3flash", .15e-6, .5e-6},
	} {
		p, err := bs.GetModelPricing(tc.model)
		require.NoError(t, err)
		require.Equal(t, tc.input, p.InputPricePerToken)
		require.Equal(t, tc.output, p.OutputPricePerToken)
	}
}
