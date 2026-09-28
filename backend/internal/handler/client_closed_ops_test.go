package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type opsAdvancedSettingsRepoStub struct {
	service.SettingRepository
	advanced string
}

func (r *opsAdvancedSettingsRepoStub) GetValue(context.Context, string) (string, error) {
	return "", service.ErrSettingNotFound
}

func (r *opsAdvancedSettingsRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{service.SettingKeyOpsAdvancedSettings: r.advanced}, nil
}

func (r *opsAdvancedSettingsRepoStub) Set(context.Context, string, string) error {
	return nil
}

// serveClientClosedRequest 以已取消的请求 context 走 failoverClientGone，复现网关标记 499 的收尾路径。
func serveClientClosedRequest(t *testing.T, ops *service.OpsService, prepare func(c *gin.Context)) {
	t.Helper()
	router := gin.New()
	router.Use(OpsErrorLoggerMiddleware(ops))
	router.POST("/v1/messages", func(c *gin.Context) {
		if prepare != nil {
			prepare(c)
		}
		failoverClientGone(c)
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/messages", nil).WithContext(ctx))
	require.Equal(t, statusClientClosedRequest, recorder.Code)
	require.Zero(t, recorder.Body.Len())
}

func TestOpsErrorLoggerMiddleware_SkipsPureClientClosed(t *testing.T) {
	setupOpsErrorLogTestQueue(t, 2)
	gin.SetMode(gin.TestMode)
	ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.True(t, ops.OpsAdvancedSettingsSnapshot().IgnoreContextCanceled)

	serveClientClosedRequest(t, ops, nil)

	require.Zero(t, OpsErrorLogQueueLength(), "纯客户端取消的 499 按 IgnoreContextCanceled 跳过")
}

func TestOpsErrorLoggerMiddleware_RecordsClientClosedAfterUpstreamFailure(t *testing.T) {
	setupOpsErrorLogTestQueue(t, 2)
	gin.SetMode(gin.TestMode)
	ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	serveClientClosedRequest(t, ops, func(c *gin.Context) {
		c.Set(service.OpsUpstreamErrorsKey, []*service.OpsUpstreamErrorEvent{{
			AccountID: 7, UpstreamStatusCode: 524, Kind: "failover", Message: "upstream timeout",
		}})
	})

	require.Equal(t, int64(1), OpsErrorLogQueueLength(), "上游失败后客户端离开仍按上游失败落库")
	job := <-opsErrorLogQueue
	require.Equal(t, statusClientClosedRequest, job.entry.StatusCode)
	require.Equal(t, "upstream", job.entry.ErrorPhase)
	require.NotNil(t, job.entry.UpstreamStatusCode)
	require.Equal(t, 524, *job.entry.UpstreamStatusCode)
}

func TestOpsErrorLoggerMiddleware_RecordsClientClosedWhenIgnoreContextCanceledDisabled(t *testing.T) {
	setupOpsErrorLogTestQueue(t, 2)
	gin.SetMode(gin.TestMode)
	settings := &opsAdvancedSettingsRepoStub{advanced: `{"ignore_context_canceled":false}`}
	ops := service.NewOpsService(nil, settings, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.False(t, ops.OpsAdvancedSettingsSnapshot().IgnoreContextCanceled)

	serveClientClosedRequest(t, ops, nil)

	require.Equal(t, int64(1), OpsErrorLogQueueLength())
	job := <-opsErrorLogQueue
	require.Equal(t, statusClientClosedRequest, job.entry.StatusCode)
}
