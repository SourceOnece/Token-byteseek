package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	billinghttp "github.com/TokenFlux/TokenRouter/internal/billing/httpapi"
	paymenthttp "github.com/TokenFlux/TokenRouter/internal/payment/httpapi"
	"github.com/TokenFlux/TokenRouter/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 匿名订单查询按 IP 限流，签名恢复与支付回调不共享这个窗口。
func TestPaymentPublicVerifyRateLimit(t *testing.T) {
	counts := &paymentVerifyCounter{values: make(map[string]int64)}
	limiter := middleware.NewRateLimiter(counts)
	pass := func(c *gin.Context) { c.Next() }
	engine := gin.New()
	mount := providePaymentRouteMount(&paymenthttp.PaymentHandler{}, &paymenthttp.PaymentWebhookHandler{}, &paymenthttp.AdminHandler{}, &billinghttp.PlanHandler{})
	mount(engine.Group("/api/v1"), httpRouteSecurity{JWT: pass, BackendUser: pass, Admin: pass, Audit: pass, Panel: middleware.NewPanelRateLimiter(limiter, nil), AuthLimiter: limiter})
	request := func(path, ip string) int {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = ip + ":12345"
		out := httptest.NewRecorder()
		engine.ServeHTTP(out, req)
		return out.Code
	}
	for i := 0; i < 20; i++ {
		require.NotEqual(t, http.StatusTooManyRequests, request("/api/v1/payment/public/orders/verify", "203.0.113.10"))
	}
	require.Equal(t, http.StatusTooManyRequests, request("/api/v1/payment/public/orders/verify", "203.0.113.10"))
	require.NotEqual(t, http.StatusTooManyRequests, request("/api/v1/payment/public/orders/verify", "203.0.113.11"))
	require.NotEqual(t, http.StatusTooManyRequests, request("/api/v1/payment/public/orders/resolve", "203.0.113.10"))
	counts.failed = true
	require.NotEqual(t, http.StatusTooManyRequests, request("/api/v1/payment/public/orders/verify", "203.0.113.10"))
}

// 装配测试只替换计数端口，实际 Redis 窗口由 infra 的存储合同测试覆盖。
type paymentVerifyCounter struct {
	values map[string]int64
	failed bool
}

func (c *paymentVerifyCounter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, int64, time.Duration, error) {
	if c.failed {
		return false, 0, 0, errors.New("counter unavailable")
	}
	c.values[key]++
	return c.values[key] <= int64(limit), c.values[key], window, nil
}
