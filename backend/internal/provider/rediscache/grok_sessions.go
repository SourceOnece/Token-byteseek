// Redis 序列化、前缀、TTL 和一次性标记继续使用已有技术 Store。
package rediscache

import (
	"github.com/TokenFlux/TokenRouter/internal/infra/redis/session"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/redis/go-redis/v9"
)

func NewGrokSessionStore(rdb *redis.Client) *provider.GrokSessionStore {
	if rdb == nil {
		return provider.NewGrokSessionStore(nil)
	}
	return provider.NewGrokSessionStore(session.New(rdb, "oauth:session:xai", provider.GrokSessionTTL))
}
