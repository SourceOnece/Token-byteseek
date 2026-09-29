package httpapi

import (
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	forward "github.com/TokenFlux/TokenRouter/internal/gateway/provider/openaiforward"
	"github.com/TokenFlux/TokenRouter/internal/protocol"
)

func openAIForwardProfile(provider *gatewayprovider.ExecutionProvider) forward.Profile {
	return forward.Profile{Platform: provider.Record.Platform, Name: provider.Record.Name, Type: provider.Record.Type, UsesCodex: provider.View().UsesOpenAICodexProtocol(), OpenAI: provider.View().IsOpenAI(), OAuth: provider.View().IsOAuth(), OAuthLike: provider.View().IsOpenAIOAuthLike(), APIKey: provider.Record.Type == capability.ProviderTypeAPIKey, Grok: provider.Record.Platform == capability.PlatformGrok, DeepSeek: provider.Record.Platform == capability.PlatformDeepseek, NativeCN: gatewayprovider.ExecutionProtocolTarget(provider).UsesNativeCNResponses(), Anthropic: gatewayprovider.ExecutionProtocolTarget(provider).IsAnthropicProtocol(), RawChat: gatewayprovider.ExecutionModelPolicy(provider).RawChat(), ResolvedChat: provider.Route.Protocol() == protocol.ProtocolOpenAIChatCompletions, Passthrough: provider.View().IsOpenAIPassthroughEnabled()}
}
