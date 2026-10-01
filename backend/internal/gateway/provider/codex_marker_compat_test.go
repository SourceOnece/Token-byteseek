package provider

import (
	"encoding/json"
	"testing"

	protocolopenai "github.com/TokenFlux/TokenRouter/internal/protocol/openai"
	"github.com/stretchr/testify/require"
)

// 只兼容历史桥接标记，不为任意请求新增 Metadata 或另一份提示词。
func TestTokenRouterTodoGuardReadCompatibility(t *testing.T) {
	for _, marker := range []string{OpenAICompatClaudeCodeTodoGuardMarker, tokenRouterClaudeCodeTodoGuardMarker} {
		require.True(t, IsOpenAICompatMessagesBridgeBody([]byte(marker)))
		body := map[string]any{"input": []any{map[string]any{"type": "message", "role": "user", "content": []any{map[string]any{"type": "input_text", "text": marker}}}}}
		require.True(t, IsOpenAICompatMessagesBridgeRequestBody(body))
		require.False(t, AppendOpenAICompatClaudeCodeTodoGuardToRequestBody(body))
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		require.True(t, IsOpenAICompatMessagesBridgeBody(encoded))
		var request protocolopenai.ResponsesRequest
		require.NoError(t, json.Unmarshal(encoded, &request))
		require.False(t, AppendOpenAICompatClaudeCodeTodoGuard(&request))
	}
}
