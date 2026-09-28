package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 取消只有同时来自实际请求和传输时才跳过；独立上游取消/超时仍保留故障证据。
func TestClientCanceledTransportGuard(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tc := range []struct {
		ctx  context.Context
		err  error
		want bool
	}{
		{canceled, &url.Error{Op: "Post", URL: "https://upstream.example", Err: context.Canceled}, true},
		{context.Background(), context.Canceled, false},
		{canceled, context.DeadlineExceeded, false},
		{canceled, errors.New("connection refused"), false},
	} {
		require.Equal(t, tc.want, isClientCanceledTransportError(tc.ctx, tc.err))
	}
	for _, openai := range []bool{false, true} {
		for _, cancelRequest := range []bool{false, true} {
			ctx := context.Background()
			if cancelRequest {
				ctx = canceled
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/responses", nil).WithContext(ctx)
			a := &Account{ID: 1, Platform: PlatformOpenAI}
			var err error
			if openai {
				err = (&OpenAIGatewayService{}).handleOpenAIUpstreamTransportError(ctx, c, a, context.Canceled, false)
			} else {
				err = (&GatewayService{}).handleUpstreamTransportError(ctx, c, a, context.Canceled, OpsUpstreamErrorEvent{})
			}
			require.ErrorIs(t, err, context.Canceled)
			_, recorded := c.Get(OpsUpstreamErrorsKey)
			require.Equal(t, !cancelRequest, recorded)
		}
	}
}
