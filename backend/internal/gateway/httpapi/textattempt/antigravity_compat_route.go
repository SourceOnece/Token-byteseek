package textattempt

import (
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
)

// shouldUseAntigravityCompat 判断提供商是否需要走 Antigravity 原生兼容桥。
func shouldUseAntigravityCompat(provider *gatewayprovider.ExecutionProvider) bool {
	return provider != nil &&
		provider.Record.Platform == capability.PlatformAntigravity &&
		provider.Record.Type == capability.ProviderTypeOAuth
}
