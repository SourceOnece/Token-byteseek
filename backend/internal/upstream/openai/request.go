package openai

import "github.com/TokenFlux/TokenRouter/internal/gateway/clientmeta"

// 复用 clientmeta 的入站客户端解析，平台许可由请求准入逻辑判断。

// IsBrowserUserAgent 委托纯客户端解析。
func IsBrowserUserAgent(userAgent string) bool { return clientmeta.IsBrowserUserAgent(userAgent) }

// IsCodexOfficialClientRequest 委托纯客户端解析。
func IsCodexOfficialClientRequest(userAgent string) bool {
	return clientmeta.IsCodexOfficialClientRequest(userAgent)
}

// IsCodexOfficialClientRequestStrict 委托纯客户端解析。
func IsCodexOfficialClientRequestStrict(userAgent string) bool {
	return clientmeta.IsCodexOfficialClientRequestStrict(userAgent)
}

// IsCodexOfficialClientOriginator 委托纯客户端解析。
func IsCodexOfficialClientOriginator(originator string) bool {
	return clientmeta.IsCodexOfficialClientOriginator(originator)
}

// IsCodexOfficialClientByHeaders 委托纯客户端解析。
func IsCodexOfficialClientByHeaders(userAgent, originator string) bool {
	return clientmeta.IsCodexOfficialClientByHeaders(userAgent, originator)
}

// PairCodexClientIdentity 委托纯客户端解析。
func PairCodexClientIdentity(userAgent string) (originator string, pairedUA string, ok bool) {
	return clientmeta.PairCodexClientIdentity(userAgent)
}

const CodexDefaultOriginator = clientmeta.CodexDefaultOriginator

// normalizeCodexClientHeader 供旧许可策略复用唯一的字符串归一化。
func normalizeCodexClientHeader(value string) string {
	return clientmeta.NormalizeCodexClientHeader(value)
}
