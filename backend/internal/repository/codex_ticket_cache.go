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

// 当前页统一读取，避免每个账号、每个模型各走一次 Redis 网络往返。
func (c *codexTicketCache) GetMany(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	if len(keys) == 0 {
		return out, nil
	}
	full := make([]string, len(keys))
	for i, key := range keys {
		full[i] = "private:codex-ticket:v1:" + key
	}
	values, err := c.client.MGet(ctx, full...).Result()
	if err != nil {
		return nil, err
	}
	for i, value := range values {
		if text, ok := value.(string); ok {
			out[keys[i]] = text
		}
	}
	return out, nil
}
func (c *codexTicketCache) Claim(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return c.client.SetNX(ctx, "private:codex-ticket-lease:v1:"+key, "1", ttl).Result()
}

// 短轮次使用唯一 owner 解锁，避免旧实例误删超时后已被别的实例取得的锁。
func (c *codexTicketCache) AcquireLease(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	// 沿用旧轮次的键空间，滚动升级时旧实例的值“1”也能阻止新实例同时采集。
	return c.client.SetNX(ctx, "private:codex-ticket-lease:v1:"+key, owner, ttl).Result()
}
func (c *codexTicketCache) ReleaseLease(ctx context.Context, key, owner string) error {
	return c.client.Eval(ctx, `if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('DEL', KEYS[1]) else return 0 end`, []string{"private:codex-ticket-lease:v1:" + key}, owner).Err()
}
