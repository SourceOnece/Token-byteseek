package app

import (
	"time"

	billinghttp "github.com/TokenFlux/TokenRouter/internal/billing/httpapi"
	paymenthttp "github.com/TokenFlux/TokenRouter/internal/payment/httpapi"
	"github.com/TokenFlux/TokenRouter/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func providePaymentRouteMount(user *paymenthttp.PaymentHandler, webhook *paymenthttp.PaymentWebhookHandler, admin *paymenthttp.AdminHandler, plans *billinghttp.PlanHandler) paymentRouteMount {
	return func(v1 *gin.RouterGroup, security httpRouteSecurity) {
		guards := paymenthttp.RouteMiddleware{JWT: security.JWT, BackendMode: security.BackendUser, Panel: security.Panel.Global(), Admin: security.Admin, Audit: security.Audit}
		// 复用现有 IP 解析和 Redis 固定窗口；缓存故障沿原 fail-open 策略。
		if security.AuthLimiter != nil {
			guards.PublicOrderVerify = security.AuthLimiter.LimitWithOptions("payment-public-order-verify", 20, time.Minute, middleware.RateLimitOptions{FailureMode: middleware.RateLimitFailOpen})
		}
		paymenthttp.RegisterRoutes(v1, user, webhook, admin, plans, guards)
	}
}
