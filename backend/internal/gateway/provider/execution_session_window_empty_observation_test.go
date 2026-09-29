package provider_test

import (
	"context"
	"net/http"
	"testing"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/stretchr/testify/require"
)

// 空响应头必须先短路；未装配健康服务的请求仍可完成正常输出与会话释放。
func TestUpdateSessionWindow_EmptyHeadersSkipsOptionalDependencies(t *testing.T) {
	var service *provideradapter.UpstreamHealth

	for name, headers := range map[string]http.Header{"nil": nil, "empty": {}, "unrelated": {"Content-Type": {"application/json"}}} {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() {
				gatewayprovider.ObserveExecutionSessionWindow(context.Background(), service, nil,

					headers)
			})
		})
	}
}
