// 代理 HTTP 接口与 DTO 包共享展示类型。
package httpapi

import (
	proxydto "github.com/TokenFlux/TokenRouter/internal/egress/httpapi/dto"
)

type AdminProxy = proxydto.AdminProxy

type AdminProxyWithProviderCount = proxydto.AdminProxyWithProviderCount

type ProxyProviderSummary = proxydto.ProxyProviderSummary
