package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 已取消的下游不得再收到人工追加的 response.failed。
func TestOpenAIEnsureForwardErrorResponse_SkipsCanceledClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	ctx, cancel := context.WithCancel(context.Background())
	c.Request = httptest.NewRequest(http.MethodPost, EndpointResponses, nil).WithContext(ctx)
	cancel()
	h := &OpenAIGatewayHandler{}
	require.False(t, h.ensureForwardErrorResponse(c, true))
	require.Equal(t, statusClientClosedRequest, c.Writer.Status())
	require.Empty(t, recorder.Body.String())
}
