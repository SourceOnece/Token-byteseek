package repository

import (
	"context"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"sync"
)

const channelCachePubSubKey = "channel_cache_updated"

// channelCacheBus 的订阅生命周期与应用同步，不依赖 Redis 关闭来退出。
type channelCacheBus struct {
	rdb     *redis.Client
	mu      sync.Mutex
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	stopped bool
}

func NewChannelCache(rdb *redis.Client) service.ChannelCachePubSub { return &channelCacheBus{rdb: rdb} }

func (b *channelCacheBus) NotifyUpdate(ctx context.Context) error {
	return b.rdb.Publish(ctx, channelCachePubSubKey, "refresh").Err()
}

func (b *channelCacheBus) SubscribeUpdates(ctx context.Context, handler func()) {
	b.mu.Lock()
	if b.cancel != nil || b.stopped {
		b.mu.Unlock()
		return
	}
	ctx, b.cancel = context.WithCancel(ctx)
	b.wg.Add(1)
	b.mu.Unlock()
	go func() {
		defer b.wg.Done()
		sub := b.rdb.Subscribe(ctx, channelCachePubSubKey)
		defer sub.Close()
		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case message, ok := <-ch:
				if !ok {
					if ctx.Err() == nil {
						slog.Warn("channel cache subscription stopped")
					}
					return
				}
				if message != nil {
					handler()
				}
			}
		}
	}()
}

func (b *channelCacheBus) StopSubscription() {
	b.mu.Lock()
	b.stopped = true
	if b.cancel != nil {
		b.cancel()
	}
	b.mu.Unlock()
	b.wg.Wait()
}
