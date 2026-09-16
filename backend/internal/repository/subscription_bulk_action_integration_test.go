//go:build integration

package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	adminhandler "github.com/TokenFlux/TokenRouter/internal/handler/admin"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 写入到期日之后注入故障，验证失败项整个事务回滚而非仅返回失败文案。
type failingBatchStatusRepo struct {
	service.UserSubscriptionRepository
	failID int64
}

func (r failingBatchStatusRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	if id == r.failID {
		return errors.New("injected database failure")
	}
	return r.UserSubscriptionRepository.UpdateStatus(ctx, id, status)
}

func TestSubscriptionBulkActionForkTransactionsAndReplay(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateUser(t, client, &service.User{Email: "subscription-batch@example.test"})
	plan, err := client.SubscriptionPlan.Create().SetName("batch-plan").SetPrice(10).Save(ctx)
	require.NoError(t, err)
	now := time.Now().Truncate(time.Second)
	first := mustCreateSubscription(t, client, &service.UserSubscription{
		UserID: user.ID, PlanID: plan.ID, StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(10 * 24 * time.Hour),
		DailyUsageUSD: 2, WeeklyUsageUSD: 5, MonthlyUsageUSD: 8,
	})
	later := mustCreateSubscription(t, client, &service.UserSubscription{
		UserID: user.ID, PlanID: plan.ID, StartsAt: first.ExpiresAt, ExpiresAt: first.ExpiresAt.Add(30 * 24 * time.Hour), Status: service.SubscriptionStatusPending,
	})
	repo := NewUserSubscriptionRepository(client)
	svc := service.NewSubscriptionService(nil, repo, nil, client, nil)
	idem := service.NewIdempotencyCoordinator(NewIdempotencyRepository(client, integrationDB), service.DefaultIdempotencyConfig())
	previous := service.DefaultIdempotencyCoordinator()
	service.SetDefaultIdempotencyCoordinator(idem)
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(previous) })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/bulk", adminhandler.NewSubscriptionHandler(svc).BulkAction)
	body, err := json.Marshal(service.BulkSubscriptionActionInput{SubscriptionIDs: []int64{first.ID, 999999999, first.ID}, Action: "extend", Days: 7})
	require.NoError(t, err)
	request := func(key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/bulk", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	require.Equal(t, 400, request("").Code, "观察模式也不能绕过新批量入口的操作键")
	key := "fork-subscription-batch-" + now.Format("150405.000000000")
	firstResponse := request(key)
	require.Equal(t, 200, firstResponse.Code, firstResponse.Body.String())
	var response struct {
		Data service.BulkSubscriptionActionResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(firstResponse.Body.Bytes(), &response))
	require.Equal(t, 1, response.Data.SuccessCount)
	require.Equal(t, 1, response.Data.FailedCount)
	require.Len(t, response.Data.Results, 2)
	replay := request(key)
	require.Equal(t, 200, replay.Code, replay.Body.String())
	require.Equal(t, "true", replay.Header().Get("X-Idempotency-Replayed"))
	require.JSONEq(t, firstResponse.Body.String(), replay.Body.String())
	changed, err := repo.GetByID(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, first.ExpiresAt.AddDate(0, 0, 7).Unix(), changed.ExpiresAt.Unix())
	chained, err := repo.GetByID(ctx, later.ID)
	require.NoError(t, err)
	require.Equal(t, changed.ExpiresAt.Unix(), chained.StartsAt.Unix())
	require.Equal(t, service.SubscriptionStatusPending, chained.Status)

	failedSvc := service.NewSubscriptionService(nil, failingBatchStatusRepo{repo, first.ID}, nil, client, nil)
	failed, err := failedSvc.BulkSubscriptionAction(ctx, &service.BulkSubscriptionActionInput{SubscriptionIDs: []int64{first.ID}, Action: "extend", Days: 3})
	require.NoError(t, err)
	require.Equal(t, 1, failed.FailedCount)
	require.Equal(t, "internal error", failed.Results[0].Error)
	afterFailure, err := repo.GetByID(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, changed.ExpiresAt, afterFailure.ExpiresAt)
	afterChainFailure, err := repo.GetByID(ctx, later.ID)
	require.NoError(t, err)
	require.Equal(t, chained.StartsAt, afterChainFailure.StartsAt)
	require.Equal(t, chained.ExpiresAt, afterChainFailure.ExpiresAt)

	reset, err := svc.BulkSubscriptionAction(ctx, &service.BulkSubscriptionActionInput{SubscriptionIDs: []int64{first.ID}, Action: "reset_quota", Daily: true})
	require.NoError(t, err)
	require.Equal(t, 1, reset.SuccessCount)
	resetSub, err := repo.GetByID(ctx, first.ID)
	require.NoError(t, err)
	require.Zero(t, resetSub.DailyUsageUSD)
	require.Equal(t, float64(5), resetSub.WeeklyUsageUSD)
	require.Equal(t, float64(8), resetSub.MonthlyUsageUSD)

	revoked, err := svc.BulkSubscriptionAction(ctx, &service.BulkSubscriptionActionInput{SubscriptionIDs: []int64{later.ID}, Action: "revoke"})
	require.NoError(t, err)
	require.Equal(t, 1, revoked.SuccessCount)
	_, err = repo.GetByID(ctx, later.ID)
	require.ErrorIs(t, err, service.ErrSubscriptionNotFound)
	restored, err := svc.BulkSubscriptionAction(ctx, &service.BulkSubscriptionActionInput{SubscriptionIDs: []int64{later.ID}, Action: "restore"})
	require.NoError(t, err)
	require.Equal(t, 1, restored.SuccessCount)
	restoredSub, err := repo.GetByID(ctx, later.ID)
	require.NoError(t, err)
	require.Equal(t, service.SubscriptionStatusPending, restoredSub.Status)
	require.Equal(t, chained.StartsAt, restoredSub.StartsAt)
}
