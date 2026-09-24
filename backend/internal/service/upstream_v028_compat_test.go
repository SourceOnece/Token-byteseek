//go:build unit

package service

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 目录缺少新型号时，不得借用旧型号价格；运营配置仍然优先。
func TestV028NewModelPricingIsolation(t *testing.T) {
	catalog := newStubPricingServiceFromJSON(t, `{"gpt-5.4":{"input_cost_per_token":9,"output_cost_per_token":9},"claude-opus-5":{"input_cost_per_token":8,"output_cost_per_token":8}}`)
	svc := NewBillingService(nil, catalog)
	for _, tt := range []struct {
		model         string
		input, output float64
	}{
		{"gpt-6-sol", 2e-6, 10e-6},
		{"gpt-6-luna-high", .1e-6, .5e-6},
		{"claude-opus-5-5", 4e-6, 20e-6},
		{"anthropic/claude-opus-5.5", 4e-6, 20e-6},
		{"grok-4.7", 2e-6, 6e-6},
	} {
		t.Run(tt.model, func(t *testing.T) {
			price, err := svc.GetModelPricing(tt.model)
			require.NoError(t, err)
			require.InDelta(t, tt.input, price.InputPricePerToken, 1e-12)
			require.InDelta(t, tt.output, price.OutputPricePerToken, 1e-12)
		})
	}
	zero := 0.0
	price, err := svc.GetModelPricingWithChannel("gpt-6-sol", &ChannelModelPricing{CacheWritePrice: &zero})
	require.NoError(t, err)
	require.Zero(t, price.CacheCreationPricePerToken)
	// 新版不能把渠道显式免费缓存又提升为标准输入价的 1.25 倍。
	price = svc.applyModelSpecificPricingPolicy("gpt-6-sol", price)
	require.Zero(t, price.CacheCreationPricePerToken)
}

func TestV028ReasoningUsesMappedModelAndPreservesAstra(t *testing.T) {
	body := []byte(`{"model":"my-alias","reasoning":{"mode":"pro","effort":"high"},"temperature":1,"include":["reasoning.encrypted_content","message.output_text.logprobs"]}`)
	out, changed, err := normalizeOpenAIResponsesReasoningMode(body, "gpt-6-sol")
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, "pro", gjson.GetBytes(out, "reasoning.mode").String())
	require.False(t, gjson.GetBytes(out, "temperature").Exists())
	require.Equal(t, `["reasoning.encrypted_content"]`, gjson.GetBytes(out, "include").Raw)
	for _, model := range []string{"gpt-6-astra", "gpt-6-luna"} {
		none := []byte(`{"model":"alias","reasoning":{"mode":"pro","effort":"none"},"temperature":1}`)
		preserved, _, err := normalizeOpenAIResponsesReasoningMode(none, model)
		require.NoError(t, err)
		require.JSONEq(t, string(none), string(preserved))
	}
}

func TestV028CodexTurnMetadataASCIIAndLossless(t *testing.T) {
	original := map[string]any{"text": "中文😀", "thread_id": "unchanged", "nested": map[string]any{"name": "测试"}}
	encoded, err := marshalCodexTurnMetadata(original)
	require.NoError(t, err)
	for _, b := range encoded {
		require.Less(t, b, byte(127))
	}
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Equal(t, original, decoded)
}

func TestV028Cloudflare1010DoesNotDisableAccount(t *testing.T) {
	for _, platform := range []string{PlatformOpenAI, PlatformOpenCodeGo} {
		h := newOpenAI403TestHarness(t, 88028, 1)
		h.account.Platform, h.account.Type = platform, AccountTypeAPIKey
		require.False(t, h.handle("error code: 1010"))
		h.requireNoAccountPenalty(t)
		// 真正的权限错误继续计数、冷却，不能因这项豁免被放行。
		require.True(t, h.handle(`{"error":{"message":"forbidden"}}`))
		require.Equal(t, 1, h.counter.increments)
	}
	require.Equal(t, http.StatusServiceUnavailable, openAIStreamFailureStatus([]byte(`{"error":{"status":503}}`), ""))
}
