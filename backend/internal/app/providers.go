package app

import (
	"context"
	"time"

	redisinfra "github.com/TokenFlux/TokenRouter/internal/infra/redis"

	"github.com/TokenFlux/TokenRouter/ent"
	"github.com/TokenFlux/TokenRouter/internal/app/bootstrap"
	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/redis/go-redis/v9"
)

func provideEnt(ctx context.Context, cfg *config.Config, manager *lifecycle.Manager) (*ent.Client, error) {
	client, _, err := bootstrap.InitEnt(ctx, cfg)
	if err != nil {
		return nil, err
	}
	manager.Register(lifecycle.Hook{Name: "Ent", StartOrder: -100, StopOrder: 910, Stop: func(context.Context) error { return client.Close() }})
	return client, nil
}

func provideRedis(ctx context.Context, cfg *config.Config, manager *lifecycle.Manager) (*redis.Client, error) {
	client := bootstrap.InitRedis(cfg)
	// 运行状态迁移完成前不装配后台任务或开放流量，避免旧键遗留造成限额重置。
	migrationCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	if err := redisinfra.MigrateProviderNames(migrationCtx, client); err != nil {
		_ = client.Close()
		return nil, err
	}
	manager.Register(lifecycle.Hook{Name: "Redis", StartOrder: -90, StopOrder: 900, Stop: func(context.Context) error { return client.Close() }})
	return client, nil
}
