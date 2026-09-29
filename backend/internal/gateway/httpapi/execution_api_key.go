package httpapi

import (
	"github.com/TokenFlux/TokenRouter/internal/apikey"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
)

// GetExecutionAPIKey 读取既有认证投影，不改变入口的鉴权和正文读取顺序。
func GetExecutionAPIKey(c interface{ Get(string) (any, bool) }) *apikey.APIKey {
	if c == nil {
		return nil
	}
	v, exists := c.Get("api_key")
	if !exists {
		return nil
	}
	apiKey, _ := v.(*apikey.APIKey)
	return apiKey
}

func OpenAIClientPolicyForbiddenMessage(result providercore.CodexClientRestrictionDetectionResult) string {
	// 按策略返回更明确的拒绝原因，同时保留旧 codex_cli_only 测试和客户端提示语义。
	if result.Policy == providercore.OpenAIOAuthClientPolicyCodexOnly {
		return "This provider only allows Codex official clients"
	}
	if result.Policy == providercore.OpenAIOAuthClientPolicyTLSRouterMatchedOnly {
		return "This provider only allows clients matched by the configured TLS router"
	}
	return "This provider only allows configured OpenAI OAuth clients"
}
