package provider

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
)

// 编程上游采用已有客户端标识，后续显式 Header 覆写仍优先；不修改其它平台。
func ApplyCompatibleClientUserAgent(value *provider.Record, target string, headers http.Header) {
	if headers == nil {
		return
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return
	}
	agent := ""
	if strings.EqualFold(parsed.Scheme, "https") && strings.EqualFold(parsed.Hostname(), "api.commandcode.ai") {
		agent = openai.CodexCanonicalUserAgent()
	} else if (value != nil && value.IsOpenCodeGo()) || (strings.EqualFold(parsed.Scheme, "https") && strings.EqualFold(parsed.Hostname(), "opencode.ai")) {
		agent = "opencode/1.0.0"
	}
	if agent == "" {
		return
	}
	for name := range headers {
		if strings.EqualFold(name, "User-Agent") {
			delete(headers, name)
		}
	}
	headers.Set("User-Agent", agent)
}
