// 本文件将唯一 Claude 授权实例绑定到原代理读取和平台客户端。
package app

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

func provideClaudeAuthorization(proxies egress.ProxyRepository, client provider.ClaudeOAuthClient) *provider.ClaudeAuthorization {
	options := provideradapter.ClaudeAuthorizationOptions(func(ctx context.Context, id int64) (string, bool) {
		proxy, err := proxies.GetByID(ctx, id)
		if err != nil || proxy == nil {
			return "", false
		}
		return proxy.URL(), true
	})
	return provider.NewClaudeAuthorization(client, options)
}
