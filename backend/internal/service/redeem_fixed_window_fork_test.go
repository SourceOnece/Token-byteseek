//go:build unit

package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPublicRedeemFixedWindowAllowsLastAttempt(t *testing.T) {
	cache := &paymentFulfillmentRedeemCacheStub{count: redeemMaxFailedAttempts - 1}
	svc := NewRedeemService(&paymentOrderLifecycleRedeemRepo{}, nil, nil, cache, nil, newPaymentConfigServiceTestClient(t), nil, nil)
	_, err := svc.Redeem(context.Background(), 42, "missing")
	require.ErrorIs(t, err, ErrRedeemCodeNotFound)
	require.Equal(t, 30, cache.count)
	// 最后一次输错记账，但下一次先限流，不再读取兑换码或取得锁。
	_, err = svc.Redeem(context.Background(), 42, "missing")
	require.ErrorIs(t, err, ErrRedeemRateLimited)
	require.Equal(t, 1, cache.acquireCalls)
	require.Equal(t, 1, cache.incrementCalls)
}

func TestPublicRedeemFixedWindowDoesNotCountSuccess(t *testing.T) {
	client := newPaymentConfigServiceTestClient(t)
	code := &RedeemCode{ID: 99, Code: "SUCCESS", Type: RedeemTypeBalance, Value: 1, Status: StatusUnused, MaxUses: 3}
	repo := &paymentOrderLifecycleRedeemRepo{codesByCode: map[string]*RedeemCode{code.Code: code}}
	users := &mockUserRepo{getByIDUser: &User{ID: 42}}
	users.updateBalanceFn = func(context.Context, int64, float64) error { return nil }
	cache := &paymentFulfillmentRedeemCacheStub{count: 7}
	svc := NewRedeemService(repo, users, nil, cache, nil, client, nil, nil)
	_, err := svc.Redeem(context.Background(), 42, code.Code)
	require.NoError(t, err)
	require.Equal(t, 7, cache.count)
	require.Zero(t, cache.incrementCalls)
	require.Len(t, repo.useCalls, 1)
}
