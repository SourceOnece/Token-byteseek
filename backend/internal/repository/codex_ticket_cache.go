package repository

import (
	"context"
	"errors"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/redis/go-redis/v9"
)

// 私有加密票据仅存 Redis，独立前缀与 TTL，不进入账号导出、调度投影或数据库备份。
type codexTicketCache struct{ client *redis.Client }

func NewCodexTicketCache(client *redis.Client) service.CodexTicketCache {
	return &codexTicketCache{client: client}
}
func (c *codexTicketCache) Get(ctx context.Context, key string) (string, error) {
	value, err := c.client.Get(ctx, "private:codex-ticket:v1:"+key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return value, err
}
func (c *codexTicketCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Set(ctx, "private:codex-ticket:v1:"+key, value, ttl).Err()
}
func (c *codexTicketCache) Claim(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return c.client.SetNX(ctx, "private:codex-ticket-lease:v1:"+key, "1", ttl).Result()
}
