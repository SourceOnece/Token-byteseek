package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 所有列表消费者共用同一解析器：普通号显式别名不被透传号遮蔽，
// 透传旧映射不复活，默认模型仍受渠道/分组资格限制。
func TestPassthroughModelCatalogKeepsMixedGroupModels(t *testing.T) {
	group := int64(63)
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"model_mapping": map[string]any{"visible-alias": "provider-model"}, "model_whitelist": []any{"provider-model"}}},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"model_mapping": map[string]any{"obsolete": "obsolete"}}, Extra: map[string]any{"openai_passthrough": true}},
	}
	svc := &GatewayService{accountRepo: &modelsListAccountRepoStub{byGroup: map[int64][]Account{group: accounts}}}
	got := svc.ResolveRequestableModels(context.Background(), &group, PlatformOpenAI)
	ids := RequestableModelIDs(got.Models)
	require.Contains(t, ids, "visible-alias")
	require.Contains(t, ids, "gpt-6-astra")
	require.NotContains(t, ids, "obsolete")
	market := (&ModelMarketplaceService{gatewayService: svc}).resolveGroupModelsWithAccounts(context.Background(), &Group{ID: group, Platform: PlatformOpenAI, ModelAllowlist: GroupModelAllowlist{Enabled: true, Models: []string{"visible-alias"}}}, accounts)
	require.Len(t, market, 1)
	require.Equal(t, "visible-alias", market[0].ID)
	// 模型限流不能因“补默认目录”绕过。
	accounts[1].Extra[modelRateLimitsKey] = map[string]any{"gpt-6-astra": map[string]any{"rate_limit_reset_at": time.Now().Add(time.Hour).Format(time.RFC3339)}}
	limited := svc.resolveRequestableModelsWithAccounts(context.Background(), &group, PlatformOpenAI, nil, accounts)
	require.NotContains(t, RequestableModelIDs(limited.Models), "gpt-6-astra")
	require.Contains(t, RequestableModelIDs(limited.Models), "visible-alias")
	// 渠道限制仍使用原定价身份，不能借透传号泄漏被限制的模型。
	svc.channelService = newRequestableModelsChannelService(group, PlatformOpenAI, Channel{ID: 63, Status: StatusActive, RestrictModels: true, ModelPricing: []ChannelModelPricing{{Platform: PlatformOpenAI, Models: []string{"visible-alias"}}}})
	restricted := svc.ResolveRequestableModels(context.Background(), &group, PlatformOpenAI)
	require.Equal(t, []string{"visible-alias"}, RequestableModelIDs(restricted.Models))
}
