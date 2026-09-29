// 响应头按已编译的过滤规则复制到下游。
package provider

import (
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/egress"
)

func FilterHeaders(src http.Header, filter *egress.CompiledHeaderFilter) http.Header {
	filtered := make(http.Header, len(src))
	for key, values := range src {
		if !filter.Allows(key) {
			continue
		}
		for _, value := range values {
			filtered.Add(key, value)
		}
	}
	return filtered
}

func WriteFilteredHeaders(dst http.Header, src http.Header, filter *egress.CompiledHeaderFilter) {
	filtered := FilterHeaders(src, filter)
	for key, values := range filtered {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
