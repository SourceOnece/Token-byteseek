// OpenAI 管理路由直接绑定提供商用例，静态平台客户端规则由组合根投影。
package app

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerhttp "github.com/TokenFlux/TokenRouter/internal/provider/httpapi"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
)

func provideOpenAIProviderOAuth(auth *provider.OpenAIAuthorization, admin *provider.Admin, quota *provider.OpenAIQuotaService, recovery *provider.RecoveryService, proxies *egress.ProxyAdmin, manager *lifecycle.Manager) *providerhttp.OpenAIOAuthHandler {
	handler := providerhttp.NewOpenAIOAuthHandler(auth, admin, quota, recovery, providerhttp.OpenAIHTTPOptions{
		ClientID: openai.OAuthClientConfigByPlatform,
		ProxyURL: func(ctx context.Context, id int64) (string, bool, error) {
			proxy, err := proxies.GetProxy(ctx, id)
			if err != nil || proxy == nil {
				return "", false, err
			}
			return proxy.URL(), true, nil
		},
	})
	// 后置流程在所属额度和提供商依赖停止前完成；超时由统一生命周期报告。
	manager.Register(lifecycle.Hook{Name: "OpenAIQuotaActions", StopOrder: 14, Stop: handler.QuotaActions.StopContext})
	return handler
}
