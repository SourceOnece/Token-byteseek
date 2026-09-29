package httpx

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 严格绑定保留动态 map 和字段校验，并且不改变其他接口的 Gin 行为。
func TestBindJSONStrictIsLocalAndValidatesStructure(t *testing.T) {
	type input struct {
		Name        string         `json:"name" binding:"required"`
		Credentials map[string]any `json:"credentials"`
	}
	contextFor := func(body string) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		return c
	}
	var valid input
	require.NoError(t, BindJSONStrict(contextFor(`{"name":"ok","credentials":{"custom":true}}`), &valid))
	require.Equal(t, true, valid.Credentials["custom"])
	for _, body := range []string{`{"name":"ok","retired":false}`, `{"name":"ok"} {}`, `{"credentials":{}}`, `{"name":"ok"} trailing`} {
		require.Error(t, BindJSONStrict(contextFor(body), &input{}))
	}
	require.NoError(t, contextFor(`{"name":"ok","unrelated_unknown":true}`).ShouldBindJSON(&input{}))
}
