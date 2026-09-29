package provider

import (
	"net/http"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/egress"
)

// ApplyRequestHeaders 在调用方原定时机应用允许覆写，不碰其它请求头或共享客户端。
func ApplyRequestHeaders(headers http.Header, policy egress.EgressPolicy, wireCasing func(string) string) {
	if headers == nil {
		return
	}
	for name, value := range policy.Headers {
		for existing := range headers {
			if strings.EqualFold(existing, name) {
				delete(headers, existing)
			}
		}
		headers[wireCasing(name)] = []string{value}
	}
}
