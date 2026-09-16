package admin

import (
	"context"
	"strconv"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/pkg/response"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
)

// BulkAction 独立入口保留原单条有效期设置语义；批量 extend 表示增减天数。
func (h *SubscriptionHandler) BulkAction(c *gin.Context) {
	var input service.BulkSubscriptionActionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := input.Validate(); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	executeSubscriptionBatch(c, "admin.subscriptions.bulk_action", input, func(ctx context.Context) (any, error) {
		return h.subscriptionService.BulkSubscriptionAction(ctx, &input)
	})
}

// 两类长批次复用有界、严格幂等执行；旧无键批量分配接口仍保持兼容分支。
func executeSubscriptionBatch(c *gin.Context, scope string, payload any, execute func(context.Context) (any, error)) {
	key, err := service.NormalizeIdempotencyKey(c.GetHeader("Idempotency-Key"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// 新批量入口必须有操作键，不能随旧客户端的观察模式降级为重复执行。
	if key == "" {
		response.ErrorFrom(c, service.ErrIdempotencyKeyRequired)
		return
	}
	coordinator := service.DefaultIdempotencyCoordinator()
	if coordinator == nil {
		response.ErrorFrom(c, service.ErrIdempotencyStoreUnavail)
		return
	}
	result, err := coordinator.Execute(c.Request.Context(), service.IdempotencyExecuteOptions{
		Scope:      scope,
		ActorScope: adminActorScope(c), Method: c.Request.Method, Route: c.FullPath(),
		IdempotencyKey: key, Payload: payload, RequireKey: true,
		TTL: service.DefaultWriteIdempotencyTTL(), ExecutionTimeout: 2 * time.Minute,
		MaxStoredResponseLen: 4 * 1024 * 1024,
	}, execute)
	if err != nil {
		if retryAfter := service.RetryAfterSecondsFromError(err); retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		response.ErrorFrom(c, err)
		return
	}
	if result.Replayed {
		c.Header("X-Idempotency-Replayed", "true")
	}
	response.Success(c, result.Data)
}
