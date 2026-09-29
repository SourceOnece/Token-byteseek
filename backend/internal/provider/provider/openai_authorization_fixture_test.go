package provider

import (
	"context"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/stretchr/testify/require"
)

// 授权测试直接配置实际 Adapter，并统一释放原生会话。
func newOpenAIAuthorizationForTest(t *testing.T, proxies egress.ProxyRepository, client OpenAIOAuthClient, dependencies ...*OpenAIAuthorizationDependencies) *provider.OpenAIAuthorization {
	t.Helper()
	deps := &OpenAIAuthorizationDependencies{}
	if len(dependencies) > 0 {
		deps = dependencies[0]
	}
	deps.Proxies = proxies
	deps.Client = client
	authorization := provider.NewOpenAIAuthorization(provider.NewOpenAISessionStore(), OpenAIAuthorizationOptions(deps))
	t.Cleanup(func() { stopOpenAIAuthorizationForTest(t, authorization) })
	return authorization
}

func stopOpenAIAuthorizationForTest(t *testing.T, authorization *provider.OpenAIAuthorization) {
	t.Helper()
	require.NoError(t, authorization.StopContext(context.Background()))
}
