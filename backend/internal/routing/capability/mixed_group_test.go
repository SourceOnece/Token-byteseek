package capability

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/protocol"
	"github.com/stretchr/testify/require"
)

func TestMixedGroupAutomaticAndExplicitRoutes(t *testing.T) {
	// 相同 Chat 请求根据候选提供商选择不同的一跳路线，原生协议始终优先。
	for _, tc := range []struct {
		platform string
		target   ProtocolID
	}{
		{PlatformAnthropic, ProtocolAnthropicMessages},
		{PlatformOpenAI, ProtocolOpenAIChatCompletions},
		{PlatformGemini, ProtocolGeminiGenerateContent},
	} {
		t.Run(tc.platform, func(t *testing.T) {
			candidate := ProviderProtocols{Platform: tc.platform, Type: ProviderTypeAPIKey, Enabled: []ProtocolID{tc.target}}
			target, ok := ResolveRoute(candidate, ProtocolOpenAIChatCompletions, nil)
			require.True(t, ok)
			require.Equal(t, tc.target, target)
			_, ok = ResolveRoute(candidate, ProtocolOpenAIChatCompletions, map[ProtocolID][]ProtocolID{ProtocolOpenAIChatCompletions: {}})
			require.Equal(t, tc.platform == PlatformOpenAI, ok)
		})
	}
}

func TestMixedGroupFallbackSnapshotOwnsLists(t *testing.T) {
	input := map[ProtocolID][]ProtocolID{ProtocolOpenAIChatCompletions: {ProtocolAnthropicMessages}, ProtocolOpenAIResponses: {}}
	copy := protocol.CloneFallbacks(input)
	copy[ProtocolOpenAIChatCompletions][0] = ProtocolGeminiGenerateContent
	require.Equal(t, ProtocolAnthropicMessages, input[ProtocolOpenAIChatCompletions][0])
	_, exists := copy[ProtocolOpenAIResponses]
	require.True(t, exists)
	require.Empty(t, copy[ProtocolOpenAIResponses])
	require.NoError(t, ValidateProtocolFallbacks("", input))
	require.Error(t, ValidateProtocolFallbacks("", map[ProtocolID][]ProtocolID{"unknown": {}}))
	require.Error(t, ValidateProtocolFallbacks("", map[ProtocolID][]ProtocolID{ProtocolOpenAIChatCompletions: {ProtocolAnthropicMessages, ProtocolAnthropicMessages}}))
}
