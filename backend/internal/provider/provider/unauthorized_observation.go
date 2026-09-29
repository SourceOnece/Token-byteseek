package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/tidwall/gjson"
)

// UnauthorizedObservation 解析供应商认证错误；只有提供商规则决定是否永久停调。
func UnauthorizedObservation(body []byte, message string) provider.UnauthorizedObservation {
	code := upstream.ExtractErrorCode(body)
	return provider.UnauthorizedObservation{Message: message, TokenRevoked: code == "token_invalidated" || code == "token_revoked", PermanentlyUnauthorized: gjson.GetBytes(body, "detail").String() == "Unauthorized"}
}
