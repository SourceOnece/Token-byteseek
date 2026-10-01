package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 两种诊断头都只能由网关生成，不能被调用方伪造，也不能留在出站请求中。
func TestClientRequestIDClearsBothBrandHeaders(t *testing.T) {
	router := gin.New()
	router.Use(ClientRequestID())
	router.GET("/", func(c *gin.Context) {
		require.Empty(t, c.GetHeader("X-TokenRouter-Request-ID"))
		require.Empty(t, c.GetHeader("X-Sub2API-Request-ID"))
		c.Status(200)
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-TokenRouter-Request-ID", "fake-new")
	req.Header.Set("X-Sub2API-Request-ID", "fake-old")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	id := res.Header().Get("X-TokenRouter-Request-ID")
	require.NotEmpty(t, id)
	require.Equal(t, id, res.Header().Get("X-Sub2API-Request-ID"))
	require.NotEqual(t, "fake-new", id)
	require.NotEqual(t, "fake-old", id)
}
