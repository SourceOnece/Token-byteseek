package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// 手动与自动共享全局4槽、账号上限及单模型互斥；等待时释放已占部分，避免互相占槽死锁。
func (s *CodexTicketService) acquireTicketCollectionSlot(ctx context.Context, cfg *codexTicketConfig, id int64, modelKey string) (func(), error) {
	owner := uuid.NewString()
	releaseKeys := func(keys []string) {
		c, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		for _, key := range keys {
			_ = s.cache.ReleaseLease(c, key, owner)
		}
	}
	for {
		if ctx.Err() != nil || !s.ticketConfigCurrent(cfg) {
			return nil, errors.New("collection_cancelled")
		}
		keys := []string{}
		modelLease := "model-work:" + modelKey
		ok, err := s.cache.AcquireLease(ctx, modelLease, owner, 45*time.Second)
		if err != nil {
			return nil, err
		}
		if ok {
			keys = append(keys, modelLease)
			groups := []struct {
				prefix string
				limit  int
			}{
				{fmt.Sprintf("account-slot:%d:", id), cfg.collectionConcurrency()},
				{"global-slot:", 4},
			}
			for _, group := range groups {
				acquired := false
				for slot := 0; slot < group.limit; slot++ {
					key := fmt.Sprintf("%s%d", group.prefix, slot)
					ok, err := s.cache.AcquireLease(ctx, key, owner, 45*time.Second)
					if err != nil {
						releaseKeys(keys)
						return nil, err
					}
					if ok {
						keys = append(keys, key)
						acquired = true
						break
					}
				}
				if !acquired {
					break
				}
			}
			if len(keys) == 3 {
				return func() { releaseKeys(keys) }, nil
			}
			releaseKeys(keys)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}
