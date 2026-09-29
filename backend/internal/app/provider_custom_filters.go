package app

import (
	"context"
	"errors"
	"github.com/TokenFlux/TokenRouter/internal/codexticket"
	"github.com/TokenFlux/TokenRouter/internal/quality"
	"github.com/TokenFlux/TokenRouter/internal/server/httpx"
	"github.com/gin-gonic/gin"
)

func withProviderCustomFilters(ctx context.Context, q, ticket string) (context.Context, error) {
	if !quality.ValidCodexQualityFilter(q) || !codexticket.ValidCodexTicketFilter(ticket) {
		return nil, errors.New("无效的检测或票据筛选")
	}
	return codexticket.WithCodexTicketFilter(quality.WithCodexQualityFilter(ctx, q), ticket), nil
}

// 请求级筛选同时供分页、全选和导出使用，不能退化成浏览器本页过滤。
func providerCustomFilters() gin.HandlerFunc {
	return func(c *gin.Context) {
		q, ticket := c.Query("quality_status"), c.Query("ticket_filter")
		if !quality.ValidCodexQualityFilter(q) || !codexticket.ValidCodexTicketFilter(ticket) {
			httpx.BadRequest(c, "无效的检测或票据筛选")
			c.Abort()
			return
		}
		ctx := quality.WithCodexQualityFilter(c.Request.Context(), q)
		ctx = codexticket.WithCodexTicketFilter(ctx, ticket)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
