//go:build wireinject

package app

import (
	redisinfra "github.com/TokenFlux/TokenRouter/internal/infra/redis"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider/rediscache"
	"github.com/google/wire"
)

// cacheProviders 直接绑定唯一缓存实现，旧调用方继续共享同一个周期任务锁实例。
var cacheProviders = wire.NewSet(
	rediscache.NewInternal500CounterCache,
	redisinfra.NewLeaderLockCache,
	wire.Bind(new(provider.CNMonitorLeader), new(*redisinfra.LeaderLock)),
)
