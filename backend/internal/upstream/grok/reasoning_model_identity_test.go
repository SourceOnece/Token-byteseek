package grok

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// TestQualifiedGrokReasoningPreservesModel 验证能力识别不会改写供应商限定 ID。
func TestQualifiedGrokReasoningPreservesModel(t *testing.T) {
	codec := BodyCodec{}
	for _, model := range []string{"grok-4.6", "xai/grok-4.6", "x-ai/grok-4.6", "grok/grok-4.6", "X-AI/GROK-4.6"} {
		t.Run(model, func(t *testing.T) {
			require.True(t, codec.GrokSupportsReasoningEffort(model))
			require.True(t, codec.GrokSupportsXHighReasoningEffort(model))
			responses, err := codec.PatchGrokResponsesBody([]byte(`{"model":"client-alias","input":"hello","reasoning":{"effort":"xhigh"}}`), model)
			require.NoError(t, err)
			require.Equal(t, model, gjson.GetBytes(responses, "model").String())
			require.Equal(t, "xhigh", gjson.GetBytes(responses, "reasoning.effort").String())

			for _, effort := range []string{"high", "xhigh"} {
				body := []byte(`{"model":"` + model + `","reasoning_effort":"` + effort + `"}`)
				chat, err := codec.NormalizeGrokChatReasoningEffort(body, model)
				require.NoError(t, err)
				require.Equal(t, model, gjson.GetBytes(chat, "model").String())
				require.Equal(t, effort, gjson.GetBytes(chat, "reasoning_effort").String())
			}
			require.Equal(t, model, NormalizeModelID(model))
			require.Equal(t, model, ResolveGrokTextResponsesModelID(model))
		})
	}
}

// TestQualifiedGrokReasoningKeepsCapabilityBounds 验证前缀不会扩大档位能力或生成隐式型号。
func TestQualifiedGrokReasoningKeepsCapabilityBounds(t *testing.T) {
	codec := BodyCodec{}
	for _, tc := range []struct {
		model string
		want  string
	}{
		{model: "xai/grok-4.5", want: "high"},
		{model: "x-ai/grok-composer-2.5-fast"},
		{model: "xai/grok-4.6-high"},
		{model: "unknown/grok-4.6"},
		{model: "xai/custom/grok-4.6"},
		{model: "xai/gpt-5.6-sol"},
	} {
		t.Run(tc.model, func(t *testing.T) {
			responses, err := codec.PatchGrokResponsesBody([]byte(`{"input":"hello","reasoning":{"effort":"xhigh"}}`), tc.model)
			require.NoError(t, err)
			require.Equal(t, tc.model, gjson.GetBytes(responses, "model").String())
			require.Equal(t, tc.want, gjson.GetBytes(responses, "reasoning.effort").String())
			chat, err := codec.NormalizeGrokChatReasoningEffort([]byte(`{"reasoning_effort":"xhigh"}`), tc.model)
			require.NoError(t, err)
			require.Equal(t, tc.want, gjson.GetBytes(chat, "reasoning_effort").String())
		})
	}
	for _, model := range []string{"xai/grok-4.6", "xai/grok-4.6-xhigh"} {
		body, err := codec.PatchGrokResponsesBody([]byte(`{"input":"hello"}`), model)
		require.NoError(t, err)
		require.False(t, gjson.GetBytes(body, "reasoning.effort").Exists())
		require.Equal(t, model, gjson.GetBytes(body, "model").String())
	}
}
