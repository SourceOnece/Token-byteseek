package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// 账号采集上限最多4，与原有全局4worker安全上限共同生效；忙时等候，不将其它模型直接跳过。
func (s *CodexTicketService) acquireTicketCollectionSlot(ctx context.Context, cfg *codexTicketConfig, id int64) (func(), error) {
	owner := uuid.NewString()
	for {
		if ctx.Err() != nil || !s.ticketConfigCurrent(cfg) {
			return nil, errors.New("collection_cancelled")
		}
		for slot := 0; slot < cfg.collectionConcurrency(); slot++ {
			key := fmt.Sprintf("account-slot:%d:%d", id, slot)
			ok, err := s.cache.AcquireLease(ctx, key, owner, 45*time.Second)
			if err != nil {
				return nil, err
			}
			if ok {
				return func() {
					c, stop := context.WithTimeout(context.Background(), time.Second)
					defer stop()
					_ = s.cache.ReleaseLease(c, key, owner)
				}, nil
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}
