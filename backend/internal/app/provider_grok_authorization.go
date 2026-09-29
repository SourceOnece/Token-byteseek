package app

import (
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider/rediscache"
	"github.com/redis/go-redis/v9"
)

// provideGrokAuthorization 保留动态配置读取与 Redis 会话替换顺序，构造不启动任务。
func provideGrokAuthorization(proxies egress.ProxyRepository, client provider.GrokAuthorizationClient, cfg *config.Config, redisClient *redis.Client) *provider.GrokAuthorization {
	options := provideradapter.GrokAuthorizationOptions(proxies, func() bool {
		return cfg != nil && cfg.Gateway.Grok.PasswordAuthEnabled
	})
	authorization := provider.NewGrokAuthorization(client, options)
	if redisClient != nil {
		authorization.Store.Stop()
		authorization.Store = rediscache.NewGrokSessionStore(redisClient)
	}
	return authorization
}
