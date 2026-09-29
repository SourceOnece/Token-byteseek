package httpapi

import (
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	protocolcore "github.com/TokenFlux/TokenRouter/internal/protocol"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/gin-gonic/gin"
)

// resolveOpenAITextProtocolForAttempt 解析当前提供商的实际文本协议，并在转发前
// 覆盖 attempt 级端点元数据，避免故障转移后沿用上一提供商的端点。
func resolveOpenAITextProtocolForAttempt(
	c *gin.Context,
	provider *gatewayprovider.ExecutionProvider,
	preferred providercore.TextProtocol,
) providercore.TextProtocol {
	// OAuth 等专用提供商始终保留既有 Responses 桥；只有 API Key 提供商参与
	// “客户端首选协议 + 路由模式 + 探测状态”的普通文本协议解析。
	protocol := providercore.TextProtocolResponses
	if provider != nil && provider.Record.Type == capability.ProviderTypeAPIKey {
		protocol = providercore.ResolveUpstreamTextProtocol(provider.Record.Extra, preferred)
	}

	if provider != nil && provider.Route.Protocol() == protocolcore.ProtocolOpenAIChatCompletions {
		protocol = providercore.TextProtocolChatCompletions
	}
	if provider != nil && provider.Route.Protocol() == protocolcore.ProtocolOpenAIResponses {
		protocol = providercore.TextProtocolResponses
	}
	endpoint := "/v1/responses"
	if protocol == providercore.TextProtocolChatCompletions {
		endpoint = "/v1/chat/completions"
	}
	SetActualOpenAIUpstreamEndpoint(c, endpoint)
	return protocol
}
