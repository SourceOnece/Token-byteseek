package provider

import (
	"context"
	"fmt"
	"time"
)

// ProviderTierManagementOptions 只转换 Drive 观测，不包含管理配置快照。
func ProviderTierManagementOptions(source *GeminiAuthorization) TierManagementOptions {
	return TierManagementOptions{Observe: func(ctx context.Context, v *Record) (GoogleOneTierObservation, error) {
		kind, ok := v.Credentials["oauth_type"].(string)
		if !ok || kind != "google_one" {
			return GoogleOneTierObservation{}, fmt.Errorf("not a google_one OAuth provider")
		}
		token, ok := v.Credentials["access_token"].(string)
		if !ok || token == "" {
			return GoogleOneTierObservation{}, fmt.Errorf("missing access_token")
		}
		var proxy string
		if v.ProxyID != nil && v.Proxy != nil {
			proxy = v.Proxy.URL()
		}
		tier, storage, err := source.FetchGoogleOneTier(ctx, token, proxy)
		out := GoogleOneTierObservation{TierID: tier}
		if storage != nil {
			out.Storage = &GoogleOneStorage{Limit: storage.Limit, Usage: storage.Usage}
			out.ObservedAt = time.Now()
		}
		return out, err
	}}
}
