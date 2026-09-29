package provider

import (
	"context"
	"errors"
	"time"

	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
)

// QoderRefreshOptions 绑定站点交换与凭据投影；持久化仍由提供商刷新协调器完成。
type QoderRefreshOptions struct {
	Exchange         qoder.RefreshExchange
	Transport        QoderTransport
	Profiles         *egressprovider.TLSProfiles
	BuildCredentials func(*provider.QoderTokenInfo) map[string]any
}

// QoderTokenRefresher 只编排一次站点刷新，不持有第二份缓存或刷新锁。
type QoderTokenRefresher struct {
	options QoderRefreshOptions
}

func NewQoderTokenRefresher(options QoderRefreshOptions) *QoderTokenRefresher {
	return &QoderTokenRefresher{options: options}
}

func (r *QoderTokenRefresher) CacheKey(value *provider.Record) string {
	return provider.QoderTokenCacheKey(value)
}

func (r *QoderTokenRefresher) CanRefresh(value *provider.Record) bool {
	return provider.CanRefreshQoder(value)
}

func (r *QoderTokenRefresher) NeedsRefresh(value *provider.Record, duration time.Duration) bool {
	return provider.NeedsRefreshQoder(value, duration)
}

func (r *QoderTokenRefresher) Refresh(ctx context.Context, value *provider.Record) (map[string]any, error) {
	if !r.CanRefresh(value) {
		return nil, errors.New("not a qoder cosy provider")
	}
	input := QoderCredentialInput(value)
	exchange := r.options.Exchange
	exchange.Doer = QoderRequestDoer(value, r.options.Transport, r.options.Profiles)
	result, err := exchange.Refresh(ctx, input)
	if err != nil {
		return nil, err
	}
	patch := qoder.TokenInfoCredentials(result.Identity, input, result.Machine)
	if r.options.BuildCredentials != nil {
		token := qoder.BuildAuthorizationTokenInfoForSite(result.Identity, result.Machine, result.Site, result.Mode, time.Time{}, nil, nil)
		patch = r.options.BuildCredentials((*provider.QoderTokenInfo)(token))
	}
	return provider.MergeQoderRefreshCredentials(value.Credentials, patch, string(result.Site), result.Mode, result.ExpiresAt), nil
}
