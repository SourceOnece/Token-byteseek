package service

import (
	"net/http"
	"strings"
)

const (
	anthropicAPIKeyAuthSchemeExtraKey = "anthropic_apikey_auth_scheme"

	AnthropicAPIKeyAuthSchemeXAPIKey             = "x_api_key"
	AnthropicAPIKeyAuthSchemeAuthorizationBearer = "authorization_bearer"
)

// GetAnthropicAPIKeyAuthScheme 返回 Anthropic API Key 账号转发上游时使用的认证头方案。
func (a *Account) GetAnthropicAPIKeyAuthScheme() string {
	if a == nil || a.Type != AccountTypeAPIKey {
		return AnthropicAPIKeyAuthSchemeXAPIKey
	}
	if a.Platform != PlatformAnthropic && !a.IsMultiProtocolAPIKey() {
		return AnthropicAPIKeyAuthSchemeXAPIKey
	}

	switch strings.TrimSpace(a.GetExtraString(anthropicAPIKeyAuthSchemeExtraKey)) {
	case AnthropicAPIKeyAuthSchemeAuthorizationBearer:
		return AnthropicAPIKeyAuthSchemeAuthorizationBearer
	default:
		return AnthropicAPIKeyAuthSchemeXAPIKey
	}
}

// isOllamaCloudAnthropicAuthBaseURL 只识别本次实际端点，不能按供应商或残留快照猜测。
func isOllamaCloudAnthropicAuthBaseURL(baseURL string) bool {
	return isOllamaCloudBaseURL(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
}

// 实际 Ollama API Key 端点使用 Bearer；其它请求保留原配置和 wire 大小写。
func setAnthropicAPIKeyAuthHeader(header http.Header, account *Account, token string, baseURLs ...string) {
	if account != nil && account.Type == AccountTypeAPIKey && len(baseURLs) > 0 && isOllamaCloudAnthropicAuthBaseURL(baseURLs[0]) {
		deleteHeaderAllForms(header, "x-api-key")
		setHeaderRaw(header, "authorization", "Bearer "+token)
		return
	}
	if account.GetAnthropicAPIKeyAuthScheme() == AnthropicAPIKeyAuthSchemeAuthorizationBearer {
		setHeaderRaw(header, "authorization", "Bearer "+token)
		return
	}
	setHeaderRaw(header, "x-api-key", token)
}
