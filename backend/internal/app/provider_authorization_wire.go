//go:build wireinject

package app

import (
	identityhttp "github.com/TokenFlux/TokenRouter/internal/identity/httpapi"
	providerhttp "github.com/TokenFlux/TokenRouter/internal/provider/httpapi"
	"github.com/google/wire"
)

// providerAuthorizationHTTPProviders 绑定实际授权端口；平台旧适配随提供商能力清理。
var providerAuthorizationHTTPProviders = wire.NewSet(
	provideClaudeAuthorizationHTTP,
	provideQoderAuthorizationHTTP,
	providerhttp.NewGeminiOAuthHandler,
	providerhttp.NewAntigravityOAuthHandler,
	providerhttp.NewCodexInviteResetHandler,
	identityhttp.NewUserAttributeHandler,
)
