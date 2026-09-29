package app

import (
	providerauth "github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	providerhttp "github.com/TokenFlux/TokenRouter/internal/provider/httpapi"
)

// provideClaudeAuthorizationHTTP 直接绑定已有授权用例，不再经管理 handler 转接。
func provideClaudeAuthorizationHTTP(source *providerauth.ClaudeAuthorization) *providerhttp.ClaudeOAuthHandler {
	return providerhttp.NewClaudeOAuthHandler(source)
}

func provideQoderAuthorizationHTTP(source *provideradapter.QoderAuthorization) *providerhttp.QoderOAuthHandler {
	return providerhttp.NewQoderOAuthHandler(source)
}
