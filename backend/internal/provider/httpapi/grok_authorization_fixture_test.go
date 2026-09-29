//go:build unit

package httpapi

import (
	"context"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/stretchr/testify/require"
)

// HTTP 组合测试保留原生授权与响应断言，共享生产参数投影。
func newGrokAuthorizationForTest(proxies egress.ProxyRepository, client provider.GrokAuthorizationClient, enabled ...bool) *provider.GrokAuthorization {
	return provider.NewGrokAuthorization(client, provideradapter.GrokAuthorizationOptions(proxies, func() bool {
		return len(enabled) > 0 && enabled[0]
	}))
}

func stopGrokAuthorizationForTest(t *testing.T, authorization *provider.GrokAuthorization) {
	t.Helper()
	require.NoError(t, authorization.StopContext(context.Background()))
}
