package app

import provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

// provideUpstreamHealth 直接发布组合根已构造的唯一观测图，不再回绑旧服务。
func provideUpstreamHealth(runtime *providerHealthRuntime) *provideradapter.UpstreamHealth {
	return runtime.Observer
}
