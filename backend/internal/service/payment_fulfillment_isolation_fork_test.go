//go:build unit

package service

import (
	"context"
	dbent "github.com/TokenFlux/TokenRouter/ent"
	"github.com/TokenFlux/TokenRouter/internal/payment"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
	"time"
)

type paymentFulfillmentRedeemCacheStub struct {
	count          int
	getCalls       int
	incrementCalls int
	acquireCalls   int
	releaseCalls   int
}

type paymentFulfillmentRedeemRepo struct {
	paymentOrderLifecycleRedeemRepo
	createCalls int
}

func (r *paymentFulfillmentRedeemRepo) Create(_ context.Context, code *RedeemCode) error {
	r.createCalls++
	if r.codesByCode == nil {
		r.codesByCode = make(map[string]*RedeemCode)
	}
	cloned := *code
	cloned.ID = int64(100 + r.createCalls)
	code.ID = cloned.ID
	r.codesByCode[cloned.Code] = &cloned
	return nil
}

func (c *paymentFulfillmentRedeemCacheStub) GetRedeemAttemptCount(context.Context, int64) (int, error) {
	c.getCalls++
	return c.count, nil
}

func (c *paymentFulfillmentRedeemCacheStub) IncrementRedeemAttemptCount(context.Context, int64) error {
	c.incrementCalls++
	c.count++
	return nil
}

func (c *paymentFulfillmentRedeemCacheStub) AcquireRedeemLock(context.Context, string, time.Duration) (bool, error) {
	c.acquireCalls++
	return true, nil
}

func (c *paymentFulfillmentRedeemCacheStub) ReleaseRedeemLock(context.Context, string) error {
	c.releaseCalls++
	return nil
}

func TestExecuteBalanceFulfillmentBypassesUserRedeemRateLimit(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	ensurePaymentAuditOrderActionUniqueIndex(t, ctx, client)
	order := createPaymentFulfillmentSubscriptionOrder(t, ctx, client, OrderStatusPaid, time.Now())
	order, err := client.PaymentOrder.UpdateOneID(order.ID).
		SetOrderType(payment.OrderTypeBalance).
		ClearPlanID().
		Save(ctx)
	require.NoError(t, err)

	redeemRepo := &paymentFulfillmentRedeemRepo{}
	credited := 0.0
	userRepo := &mockUserRepo{getByIDUser: &User{ID: order.UserID, Balance: 0}}
	userRepo.updateBalanceFn = func(_ context.Context, id int64, amount float64) error {
		require.Equal(t, order.UserID, id)
		credited += amount
		return nil
	}
	cache := &paymentFulfillmentRedeemCacheStub{count: redeemMaxFailedAttempts}
	redeemService := NewRedeemService(redeemRepo, userRepo, nil, cache, nil, client, nil, nil)
	svc := &PaymentService{entClient: client, redeemService: redeemService, userRepo: userRepo}

	require.NoError(t, svc.ExecuteBalanceFulfillment(ctx, order.ID))
	require.InDelta(t, order.Amount, credited, 1e-8)
	require.Zero(t, cache.getCalls, "trusted payment fulfillment must not read the public failure counter")
	require.Zero(t, cache.incrementCalls, "trusted payment fulfillment must not mutate the public failure counter")
	require.Equal(t, 1, cache.acquireCalls)
	require.Equal(t, 1, cache.releaseCalls)
	require.Equal(t, 1, redeemRepo.createCalls)
	require.Len(t, redeemRepo.useCalls, 1)

	usedCode := redeemRepo.codesByCode[order.RechargeCode]
	require.Equal(t, StatusUsed, usedCode.Status)
	require.NotNil(t, usedCode.UsedBy)
	require.Equal(t, order.UserID, *usedCode.UsedBy)
	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, reloaded.Status)
}

func TestExecuteBalanceFulfillmentRejectsCodeUsedByAnotherUser(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	ensurePaymentAuditOrderActionUniqueIndex(t, ctx, client)
	order := createPaymentFulfillmentSubscriptionOrder(t, ctx, client, OrderStatusPaid, time.Now())
	order, err := client.PaymentOrder.UpdateOneID(order.ID).
		SetOrderType(payment.OrderTypeBalance).
		ClearPlanID().
		Save(ctx)
	require.NoError(t, err)

	otherUserID := order.UserID + 1
	redeemRepo := &redeemCodeRepoStub{codesByCode: map[string]*RedeemCode{
		order.RechargeCode: {
			ID:      101,
			Code:    order.RechargeCode,
			Type:    RedeemTypeBalance,
			Value:   order.Amount,
			Status:  StatusUsed,
			MaxUses: 1, UsedCount: 1,
			UsedBy: &otherUserID,
		},
	}}
	svc := &PaymentService{
		entClient:     client,
		redeemService: &RedeemService{redeemRepo: redeemRepo},
	}

	err = svc.ExecuteBalanceFulfillment(ctx, order.ID)
	require.ErrorContains(t, err, "payment redeem code user mismatch")
	require.Empty(t, redeemRepo.useCalls)
	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusFailed, reloaded.Status)
	require.Nil(t, reloaded.CompletedAt)
}

func TestPaymentRedeemCodeForkOwnershipAndSingleUse(t *testing.T) {
	userID, otherUserID := int64(42), int64(43)
	order := &dbent.PaymentOrder{ID: 7, UserID: userID, RechargeCode: "PAY-7", Amount: 80}
	valid := RedeemCode{Code: order.RechargeCode, Type: RedeemTypeBalance, Value: 80, Status: StatusUnused, MaxUses: 1}
	require.NoError(t, validatePaymentRedeemCode(order, &valid))
	for name, mutate := range map[string]func(*RedeemCode){
		"code":              func(c *RedeemCode) { c.Code = "OTHER" },
		"type":              func(c *RedeemCode) { c.Type = RedeemTypeConcurrency },
		"amount":            func(c *RedeemCode) { c.Value = 79 },
		"NaN":               func(c *RedeemCode) { c.Value = math.NaN() },
		"infinity":          func(c *RedeemCode) { c.Value = math.Inf(1) },
		"unlimited":         func(c *RedeemCode) { c.MaxUses = 0 },
		"multiple":          func(c *RedeemCode) { c.MaxUses = 3 },
		"unused-with-count": func(c *RedeemCode) { c.UsedCount = 1 },
		"unused-with-user":  func(c *RedeemCode) { c.UsedBy = &userID },
		"wrong-user":        func(c *RedeemCode) { c.Status = StatusUsed; c.UsedCount = 1; c.UsedBy = &otherUserID },
		"missing-user":      func(c *RedeemCode) { c.Status = StatusUsed; c.UsedCount = 1 },
		"expired":           func(c *RedeemCode) { c.Status = StatusExpired },
	} {
		t.Run(name, func(t *testing.T) {
			code := valid
			mutate(&code)
			require.Error(t, validatePaymentRedeemCode(order, &code))
		})
	}
	valid.Status = StatusUsed
	valid.UsedCount = 1
	valid.UsedBy = &userID
	require.NoError(t, validatePaymentRedeemCode(order, &valid))
}

func TestRedeemTrustedPathsDoNotChangePublicFailureCounter(t *testing.T) {
	for _, mode := range []string{"public", "admin", "payment"} {
		t.Run(mode, func(t *testing.T) {
			cache := &paymentFulfillmentRedeemCacheStub{count: redeemMaxFailedAttempts}
			svc := NewRedeemService(&paymentOrderLifecycleRedeemRepo{}, nil, nil, cache, nil, newPaymentConfigServiceTestClient(t), nil, nil)
			var err error
			switch mode {
			case "public":
				_, err = svc.Redeem(context.Background(), 42, "missing")
			case "admin":
				_, err = svc.RedeemForAdminFulfillment(context.Background(), 42, "missing")
			case "payment":
				_, err = svc.redeemForPaymentFulfillment(context.Background(), &dbent.PaymentOrder{UserID: 42, RechargeCode: "missing", Amount: 10})
			}
			if mode == "public" {
				require.ErrorIs(t, err, ErrRedeemRateLimited)
				require.Equal(t, 1, cache.getCalls)
				require.Zero(t, cache.acquireCalls)
			} else {
				require.ErrorIs(t, err, ErrRedeemCodeNotFound)
				require.Zero(t, cache.getCalls)
				require.Equal(t, 1, cache.acquireCalls)
				require.Equal(t, 1, cache.releaseCalls)
			}
			require.Zero(t, cache.incrementCalls)
		})
	}
	cache := &paymentFulfillmentRedeemCacheStub{}
	svc := NewRedeemService(&paymentOrderLifecycleRedeemRepo{}, nil, nil, cache, nil, newPaymentConfigServiceTestClient(t), nil, nil)
	_, err := svc.Redeem(context.Background(), 42, "missing")
	require.ErrorIs(t, err, ErrRedeemCodeNotFound)
	require.Equal(t, 1, cache.incrementCalls)
}

func TestPaymentRedeemValidatesLockedCodeBeforeCrediting(t *testing.T) {
	// 模拟预查询后码金额被修改，行锁内校验必须在访问用户或发放权益前拒绝。
	code := &RedeemCode{ID: 1, Code: "PAY-1", Type: RedeemTypeBalance, Value: 79, Status: StatusUnused, MaxUses: 1}
	repo := &paymentOrderLifecycleRedeemRepo{codesByCode: map[string]*RedeemCode{code.Code: code}}
	cache := &paymentFulfillmentRedeemCacheStub{count: redeemMaxFailedAttempts}
	svc := NewRedeemService(repo, nil, nil, cache, nil, newPaymentConfigServiceTestClient(t), nil, nil)
	_, err := svc.redeemForPaymentFulfillment(context.Background(), &dbent.PaymentOrder{ID: 1, UserID: 42, RechargeCode: code.Code, Amount: 80})
	require.ErrorContains(t, err, "amount mismatch")
	require.Empty(t, repo.useCalls)
	require.Zero(t, cache.incrementCalls)
}
