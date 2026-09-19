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

// 手动批次可能很长，续租只能延长自己持有的锁，丢锁立即停止后续采集。
func (c *codexTicketCache) RenewLease(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	return c.client.Eval(ctx, `if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('PEXPIRE', KEYS[1], ARGV[2]) else return 0 end`, []string{"private:codex-ticket-lease:v1:" + key}, owner, ttl.Milliseconds()).Bool()
}

// 最新展示按完成时间原子比较，迟到旧请求不得覆盖更新的采集结果；不参与调度判断。
func (c *codexTicketCache) SetLatest(ctx context.Context, key, value string, finishedUS int64, ttl time.Duration) error {
	return c.client.Eval(ctx, `
local old=redis.call('GET',KEYS[1])
if old then
 local ok, data=pcall(cjson.decode,old)
 if ok and type(data)=='table' and tonumber(data.finished_us or 0)>tonumber(ARGV[2]) then return 0 end
end
redis.call('SET',KEYS[1],ARGV[1],'PX',ARGV[3])
return 1`, []string{"private:codex-ticket:v1:latest:" + key}, value, finishedUS, ttl.Milliseconds()).Err()
}

// 同一Lua中比较本次使用的密文、去重信号和可选废票；迟到旧响应不能删掉新票。
func (c *codexTicketCache) ObserveTicket(ctx context.Context, key, expected, reason string, revoke bool, at time.Time) (bool, error) {
	return c.client.Eval(ctx, `
if redis.call('GET',KEYS[1])~=ARGV[1] then return 0 end
local previous=redis.call('GET',KEYS[2])
local state={count=0}
if previous then local ok,v=pcall(cjson.decode,previous); if ok and type(v)=='table' then state=v end end
local signature=redis.sha1hex(ARGV[1])..':'..ARGV[2]
if state.signature==signature then return 0 end
state.count=tonumber(state.count or 0)+1
state.reason=ARGV[2]; state.checked_at=ARGV[4]; state.signature=signature; state.action='observed'
if ARGV[3]=='1' then redis.call('DEL',KEYS[1]); redis.call('DEL',KEYS[3]); state.action='revoked' end
redis.call('SET',KEYS[2],cjson.encode(state),'EX',86400)
return 1`, []string{"private:codex-ticket:v1:" + key, "private:codex-ticket:v1:watchdog:" + key, "private:codex-ticket-lease:v1:probe:" + key}, expected, reason, map[bool]string{false: "0", true: "1"}[revoke], at.Format(time.RFC3339Nano)).Bool()
}
