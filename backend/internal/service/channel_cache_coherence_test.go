//go:build unit

package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 查询期间失效时，旧成功或失败结果都不得覆盖已刷新的快照。
func TestChannelCacheInvalidationDuringLoad(t *testing.T) {
	for _, oldError := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[oldError], func(t *testing.T) {
			started, release := make(chan struct{}), make(chan struct{})
			var reads atomic.Int32
			repo := &mockChannelRepository{listAllFn: func(context.Context) ([]Channel, error) {
				if reads.Add(1) == 1 {
					close(started)
					<-release
					if oldError {
						return nil, errors.New("old query failure")
					}
					return []Channel{{ID: 1, Name: "old"}}, nil
				}
				return []Channel{{ID: 1, Name: "new"}}, nil
			}}
			svc := NewChannelService(repo, nil)
			done := make(chan error, 1)
			go func() { _, err := svc.loadCache(context.Background()); done <- err }()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("query not started")
			}
			svc.clearLocalCache()
			_, err := svc.loadCache(context.Background())
			require.NoError(t, err)
			close(release)
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("query not released")
			}
			cached, err := svc.loadCache(context.Background())
			require.NoError(t, err)
			require.Equal(t, "new", cached.byID[1].Name)
		})
	}
}

type channelNotificationStub struct {
	handler func()
	sends   int
	stopped bool
}

func (b *channelNotificationStub) NotifyUpdate(context.Context) error {
	b.sends++
	b.handler()
	return nil
}
func (b *channelNotificationStub) SubscribeUpdates(_ context.Context, handler func()) {
	b.handler = handler
}
func (b *channelNotificationStub) StopSubscription() { b.stopped = true }

func TestChannelCacheNotificationDoesNotRebroadcast(t *testing.T) {
	bus := &channelNotificationStub{}
	svc := ProvideChannelService(&mockChannelRepository{}, nil, bus)
	svc.InvalidateCache()
	require.Equal(t, 1, bus.sends)
	require.Nil(t, svc.cache.Load().(*channelCache))
	svc.Stop()
	require.True(t, bus.stopped)
}
