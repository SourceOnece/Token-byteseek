package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/TokenFlux/TokenRouter/internal/upstream/antigravity"
	"github.com/TokenFlux/TokenRouter/internal/upstream/kimi"
)

// ForbiddenObservation 复用平台解析，不作提供商持久化或调度裁决。
func ForbiddenObservation(value *provider.Record, message string, body []byte) provider.ForbiddenObservation {
	kind := antigravity.ClassifyForbiddenType(string(body))
	url := ""
	if kind == provider.ForbiddenTypeValidation {
		url = antigravity.ExtractValidationURL(string(body))
	}
	return provider.ForbiddenObservation{Message: message, Body: body, HTML: upstream.IsHTMLResponse(body), Kind: kind, ValidationURL: url, ConcurrentRequestLimited: CNConcurrencyLimit403(value, message), ConcurrencyReason: "cn_concurrency_limit: " + kimi.ConcurrentRequestLimitMessage}
}

// CNConcurrencyLimit403 仅认可 Kimi 的精确限制消息，其他平台仍使用原错误处理。
func CNConcurrencyLimit403(value *provider.Record, message string) bool {
	return value != nil && value.Platform == provider.PlatformKimi && kimi.IsConcurrencyLimitMessage(message)
}
