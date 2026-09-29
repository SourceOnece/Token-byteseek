// 本文件直接装配唯一 Antigravity 授权状态与原代理读取。
package app

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

func provideAntigravityAuthorization(proxies egress.ProxyRepository) *provider.AntigravityAuthorization {
	options := provideradapter.AntigravityAuthorizationOptions(func(ctx context.Context, id int64) (string, bool) {
		proxy, err := proxies.GetByID(ctx, id)
		if err != nil || proxy == nil {
			return "", false
		}
		return proxy.URL(), true
	})
	return provider.NewAntigravityAuthorization(options)
}
