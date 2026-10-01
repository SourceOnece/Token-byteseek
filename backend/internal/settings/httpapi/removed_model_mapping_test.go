package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestRemovedCrossClientModelMappingField 验证废弃字段在读取或修改任何设置前被拒绝。
func TestRemovedCrossClientModelMappingField(t *testing.T) {
	for _, value := range []string{"true", "false", "null"} {
		writer := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(writer)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(`{"grok_cross_client_model_map_enabled":`+value+`}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		(&Handler{}).UpdateSettings(ctx)
		require.Equal(t, http.StatusBadRequest, writer.Code)
		require.Contains(t, writer.Body.String(), "model_mapping")
	}
}
