package provider_test

import (
	"context"
	"testing"
	"time"

	gatewaymedia "github.com/TokenFlux/TokenRouter/internal/gateway/media"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	"github.com/TokenFlux/TokenRouter/internal/creative"
	"github.com/TokenFlux/TokenRouter/internal/gateway/requeststate"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	protocolcore "github.com/TokenFlux/TokenRouter/internal/protocol"

	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func TestProtocolRouteNativeFirstAndExplicitFallback(t *testing.T) {
	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 1, Platform: capability.PlatformDeepseek, Type: capability.ProviderTypeAPIKey, Credentials: map[string]any{providercore.UpstreamProtocolsKey: []string{"anthropic_messages", "openai_responses"}, "api_base_urls": map[string]any{"anthropic": "https://relay.example/messages", "responses": "https://relay.example/responses"}}}}
	group := &routing.Group{AllowedProtocols: []protocolcore.ProtocolID{protocolcore.ProtocolAnthropicMessages}, ProtocolFallbacks: map[protocolcore.ProtocolID][]protocolcore.ProtocolID{protocolcore.ProtocolAnthropicMessages: {protocolcore.ProtocolOpenAIResponses}}}
	ctx := requeststate.WithClientProtocol(requeststate.WithGroup(context.Background(), group), protocolcore.ProtocolAnthropicMessages)
	selected, err := gatewayprovider.ProviderForProtocolAttempt(ctx, provider)
	require.NoError(t, err)
	require.Equal(t, providercore.APIProtocolAnthropic, gatewayprovider.ExecutionProtocolTarget(selected).GetAPIProtocol())
	require.Empty(t, provider.Route.Protocol())
	require.Equal(t, "https://relay.example/messages", gatewayprovider.ExecutionProtocolTarget(selected).GetAnthropicProtocolBaseURL())
	provider.Record.Credentials[providercore.UpstreamProtocolsKey] = []string{"openai_responses"}
	selected, err = gatewayprovider.ProviderForProtocolAttempt(ctx, provider)
	require.NoError(t, err)
	require.Equal(t, providercore.APIProtocolResponses, gatewayprovider.ExecutionProtocolTarget(selected).GetAPIProtocol())
	require.Equal(t, "https://relay.example/responses", gatewayprovider.ExecutionProtocolTarget(selected).GetCNProtocolBaseURL(providercore.APIProtocolResponses))
	// 转换目标无需向客户端开放；下一次切号重新使用该候选的集合。
	require.False(t, group.AllowsClientProtocol(protocolcore.ProtocolOpenAIResponses))
	next := *provider
	next.Record.Credentials = map[string]any{providercore.UpstreamProtocolsKey: []string{"openai_chat_completions"}}
	require.False(t, gatewayprovider.ExecutionModelPolicy(&next).AllowsProtocol(ctx))
	group.ProtocolFallbacks[protocolcore.ProtocolAnthropicMessages] = []protocolcore.ProtocolID{}
	ctx = requeststate.WithGroup(ctx, group)
	require.False(t, gatewayprovider.ExecutionModelPolicy(provider).AllowsProtocol(ctx))
}

func TestProtocolConversionProviderConstraints(t *testing.T) {
	for _, tc := range []struct {
		platform, kind, auth string
		source, target       protocolcore.ProtocolID
		want                 bool
	}{
		{capability.PlatformOpenAI, capability.ProviderTypeOAuth, "", protocolcore.ProtocolImagesEdits, protocolcore.ProtocolOpenAIResponses, true},
		{capability.PlatformOpenAI, capability.ProviderTypeAPIKey, "", protocolcore.ProtocolImagesEdits, protocolcore.ProtocolOpenAIResponses, false},
		{capability.PlatformGrok, capability.ProviderTypeAPIKey, "", protocolcore.ProtocolResponsesWebSocket, protocolcore.ProtocolOpenAIResponses, true},
		{capability.PlatformGrok, capability.ProviderTypeOAuth, "", protocolcore.ProtocolWebSearch, protocolcore.ProtocolOpenAIResponses, true},
		{capability.PlatformOpenAI, capability.ProviderTypeOAuth, providercore.OpenAIAuthModePersonalAccessToken, protocolcore.ProtocolAlphaSearch, protocolcore.ProtocolOpenAIResponses, true},
		{capability.PlatformOpenAI, capability.ProviderTypeOAuth, "", protocolcore.ProtocolAlphaSearch, protocolcore.ProtocolOpenAIResponses, false},
		{capability.PlatformOpenAI, capability.ProviderTypeAPIKey, "", protocolcore.ProtocolEmbeddings, protocolcore.ProtocolOpenAIResponses, false},
		{capability.PlatformGrok, capability.ProviderTypeAPIKey, "", protocolcore.ProtocolTTS, protocolcore.ProtocolOpenAIResponses, false},
	} {
		t.Run(string(tc.source)+"/"+tc.platform+"/"+tc.kind+"/"+tc.auth, func(t *testing.T) {
			require.Equal(t, tc.want, capability.SupportsProtocolConversion(tc.platform, tc.kind, tc.auth, tc.source, tc.target))
		})
	}
}

func TestProtocolImagePolicyAndBatchBinding(t *testing.T) {
	group := &routing.Group{AllowedProtocols: []protocolcore.ProtocolID{}, ResponsesImagePolicy: "enabled"}
	require.NoError(t, routing.NormalizeGroupProtocolPolicy(group, nil))
	*group = *routing.CloneGroup(group)
	require.False(t, gatewaymedia.GroupImagePermission(group != nil, group.AllowImageGeneration))
	require.True(t, routing.GroupAllowsResponsesImages(group))
	require.Equal(t, providercore.CodexImagePolicyAllow, gatewayprovider.GroupResponsesExplicitToolPolicy(group, providercore.CodexImagePolicyStrip))
	group.ResponsesImagePolicy = "block"
	require.Equal(t, providercore.CodexImagePolicyStrip, gatewayprovider.GroupResponsesExplicitToolPolicy(group, providercore.CodexImagePolicyAllow))
	for _, tc := range []struct {
		kind   string
		target protocolcore.ProtocolID
	}{{capability.ProviderTypeAPIKey, protocolcore.ProtocolGeminiBatch}, {capability.ProviderTypeServiceAccount, protocolcore.ProtocolVertexBatch}} {
		a := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformGemini, Type: tc.kind, Credentials: map[string]any{providercore.UpstreamProtocolsKey: []protocolcore.ProtocolID{tc.target}}}}
		target, ok := gatewayprovider.ExecutionModelPolicy(a).ProtocolRoute(nil, protocolcore.ProtocolImageBatches)
		require.True(t, ok)
		require.Equal(t, tc.target, target)
	}
}

func TestProtocolAuxiliaryModelURLAndIndependentTransports(t *testing.T) {
	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformKimi, Type: capability.ProviderTypeAPIKey, Credentials: map[string]any{"api_protocol": "anthropic", "base_url": "https://relay.example/custom/anthropic", "api_key": "test"}}}
	require.NoError(t, gatewayprovider.NormalizeExecutionProtocols(provider))
	require.Equal(t, "https://relay.example/custom", gatewayprovider.ExecutionProtocolTarget(provider).GetOpenAIFormatBaseURL())
	for _, protocol := range []protocolcore.ProtocolID{protocolcore.ProtocolResponsesWebSocket, protocolcore.ProtocolResponsesCompact} {
		a := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey, Credentials: map[string]any{providercore.UpstreamProtocolsKey: []protocolcore.ProtocolID{protocol}}}}
		ctx := requeststate.WithClientProtocol(context.Background(), protocol)
		require.True(t, gatewayprovider.
			SupportsRequestCapability(ctx, a, providercore.OpenAIEndpointCapabilityResponses))
		require.False(t, provideradapter.SupportsOpenAIEndpoint(gatewayprovider.ExecutionProtocolRecord(a), providercore.OpenAIEndpointCapabilityTextGeneration))
	}
	group := &routing.Group{AllowedProtocols: []protocolcore.ProtocolID{protocolcore.ProtocolImagesEdits}, ResponsesImagePolicy: "block"}
	require.Equal(t, []string{creative.CreativeOperationEdit, creative.CreativeOperationInpaint}, creative.OperationsForGroup(group.ResponsesImagePolicy != "" || group.ProtocolFallbacks != nil, group.AllowsClientProtocol)[creative.PlatformOpenAI])
	require.Nil(t, gatewayprovider.ResponsesPolicyGroup(requeststate.WithClientProtocol(context.Background(), protocolcore.ProtocolImagesEdits), group))
}
