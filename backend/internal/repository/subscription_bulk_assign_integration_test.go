//go:build integration

package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	adminhandler "github.com/TokenFlux/TokenRouter/internal/handler/admin"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionBulkAssignReplayKeepsPlanChain(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	user := mustCreateUser(t, client, &service.User{Email: "assign-batch@example.test"})
	plan, err := client.SubscriptionPlan.Create().SetName("assign-batch-plan").SetPrice(5).SetValidityDays(30).Save(ctx)
	require.NoError(t, err)
	repo := NewUserSubscriptionRepository(client)
	svc := service.NewSubscriptionService(nil, repo, nil, client, nil)
	initial, err := svc.AssignSubscription(ctx, &service.AssignSubscriptionInput{UserID: user.ID, PlanID: plan.ID, ValidityDays: 10})
	require.NoError(t, err)
	previous := service.DefaultIdempotencyCoordinator()
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(NewIdempotencyRepository(client, integrationDB), service.DefaultIdempotencyConfig()))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(previous) })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/bulk-assign", adminhandler.NewSubscriptionHandler(svc).BulkAssign)
	payload, err := json.Marshal(adminhandler.BulkAssignSubscriptionRequest{UserIDs: []int64{user.ID, 999999999, user.ID}, PlanID: plan.ID, ValidityDays: 7})
	require.NoError(t, err)
	key := uuid.NewString()
	send := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/bulk-assign", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	first := send()
	require.Equal(t, 200, first.Code, first.Body.String())
	var result struct {
		Data struct {
			Success int `json:"success_count"`
			Failed  int `json:"failed_count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &result))
	require.Equal(t, 1, result.Data.Success)
	require.Equal(t, 1, result.Data.Failed)
	replay := send()
	require.Equal(t, 200, replay.Code, replay.Body.String())
	require.Equal(t, "true", replay.Header().Get("X-Idempotency-Replayed"))
	require.JSONEq(t, first.Body.String(), replay.Body.String())
	chain, err := repo.ListByUserIDAndPlanID(ctx, user.ID, plan.ID)
	require.NoError(t, err)
	require.Len(t, chain, 2)
	require.Equal(t, initial.ExpiresAt.Unix(), chain[1].StartsAt.Unix())
	require.Equal(t, initial.ExpiresAt.AddDate(0, 0, 7).Unix(), chain[1].ExpiresAt.Unix())
}
