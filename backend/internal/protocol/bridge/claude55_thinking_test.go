package bridge

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

// 5.5 经 Responses 往返时保留上游签名和正文顺序，不能把签名伪造成思考文字。
func TestClaude55SignedThinkingRoundTrip(t *testing.T) {
	response := &AnthropicResponse{ID: "msg_fixture", Model: "claude-sonnet-5-5", Content: []AnthropicContentBlock{{Type: "text", Text: "前文"}, {Type: "thinking", Thinking: "思考", Signature: "fixture-signature"}, {Type: "redacted_thinking", Data: "opaque-fixture"}, {Type: "text", Text: "后文"}}}
	out := AnthropicToResponsesResponse(testRuntime(), response)
	require.Len(t, out.Output, 4)
	require.Equal(t, "message", out.Output[0].Type)
	require.Equal(t, "reasoning", out.Output[1].Type)
	input, err := json.Marshal(out.Output)
	require.NoError(t, err)
	req, err := ResponsesToAnthropicRequest(&ResponsesRequest{Model: response.Model, Input: input, Reasoning: &ResponsesReasoning{Effort: "high"}})
	require.NoError(t, err)
	require.Equal(t, "adaptive", req.Thinking.Type)
	raw, _ := json.Marshal(req.Messages)
	require.Contains(t, string(raw), "fixture-signature")
	require.Contains(t, string(raw), "opaque-fixture")
	block, ok := decodeAnthropicThinking("unknown-openai-ciphertext")
	require.False(t, ok)
	require.Empty(t, block.Type)
}

func TestSonnet55BetweenToolsAndValidation(t *testing.T) {
	req := &ResponsesRequest{Model: "claude-sonnet-5-5", Input: json.RawMessage(`"hello"`), Reasoning: &ResponsesReasoning{Effort: "none"}}
	out, err := ResponsesToAnthropicRequest(req)
	require.NoError(t, err)
	require.Equal(t, "between_tools", out.Thinking.Type)
	require.Equal(t, "low", out.OutputConfig.Effort)
	req.ToolChoice = json.RawMessage(`"required"`)
	_, err = ResponsesToAnthropicRequest(req)
	require.Error(t, err)
	req.ToolChoice = nil
	temp := 0.5
	req.Temperature = &temp
	_, err = ResponsesToAnthropicRequest(req)
	require.Error(t, err)
}
