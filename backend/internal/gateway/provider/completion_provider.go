package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/provider"

	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	"github.com/TokenFlux/TokenRouter/internal/gateway/completion"
)

func ProjectCompletionProvider(v *provider.Record) *completion.ProviderSnapshot {
	if v == nil {
		return nil
	}
	out := &completion.ProviderSnapshot{
		ID:                         v.ID,
		CacheTTLOverrideEnabled:    v.IsCacheTTLOverrideEnabled(),
		CacheTTLOverrideTarget:     v.GetCacheTTLOverrideTarget(),
		AnthropicOAuthOrSetupToken: v.IsAnthropicOAuthOrSetupToken(),
		Type:                       v.Type,
		Platform:                   v.Platform,
		RateMultiplier:             v.BillingRateMultiplier(),
		OpenAI:                     v.IsOpenAI(),
		CNProvider:                 v.IsCNProvider(),
		OAuthLike:                  v.IsOpenAIOAuthLike(),
		QuotaEligible:              v.IsAPIKeyOrBedrock(),
		HasQuotaLimit:              v.HasAnyQuotaLimit(),
		CredentialProviderID:       v.ParentProviderID,
		Notification:               provideradapter.QuotaNotification(&provider.Record{ID: v.ID, Name: v.Name, Platform: v.Platform, Type: v.Type, Extra: v.Extra}),
	}
	return completion.SnapshotProvider(out)
}
