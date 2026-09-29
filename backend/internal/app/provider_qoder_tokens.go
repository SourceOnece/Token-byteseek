package app

import (
	"log/slog"

	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
)

// provideQoderTokens 构造唯一提供商会话缓存，站点交换由供应商构建器负责。
func provideQoderTokens(transport provideradapter.QoderTransport, profiles *egressprovider.TLSProfiles) *provideradapter.QoderTokenProvider {
	source := provideradapter.NewQoderTokenProvider(qoder.SessionBuilder{})
	source.SetHTTPUpstream(transport, profiles)
	return source
}

// provideTokenCacheInvalidator 复用已登记会话和令牌缓存，不创建第二份状态。
func provideTokenCacheInvalidator(cache provider.AccessTokenCache, sessions *provideradapter.QoderTokenProvider) provider.TokenCacheInvalidator {
	return provider.NewCompositeTokenCacheInvalidator(cache, sessions, slog.Warn)
}
