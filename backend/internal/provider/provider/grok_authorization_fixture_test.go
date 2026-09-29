//go:build unit

package provider

import (
	"context"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/stretchr/testify/require"
)

// 测试直接构造实际授权拥有者，配置只投影密码功能开关。
func newGrokAuthorizationForTest(proxies egress.ProxyRepository, client provider.GrokAuthorizationClient, enabled ...bool) *provider.GrokAuthorization {
	options := GrokAuthorizationOptions(proxies, func() bool { return len(enabled) > 0 && enabled[0] })
	return provider.NewGrokAuthorization(client, options)
}

func stopGrokAuthorizationForTest(t *testing.T, authorization *provider.GrokAuthorization) {
	t.Helper()
	require.NoError(t, authorization.StopContext(context.Background()))
}
