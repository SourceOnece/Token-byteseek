package apicompat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 驱动工具块经过 Anthropic→Responses 转换，收集按顺序发出的事件。
func collectToolCallStreamEvents(t *testing.T, blockInput json.RawMessage, partials []string) []ResponsesStreamEvent {
	t.Helper()

	state := NewAnthropicEventToResponsesState()
	var all []ResponsesStreamEvent

	all = append(all, AnthropicEventToResponsesEvents(&AnthropicStreamEvent{
		Type: "message_start",
		Message: &AnthropicResponse{
			ID:    "msg_tool_input",
			Model: "gemini-3.7-flash",
			Usage: AnthropicUsage{InputTokens: 12},
		},
	}, state)...)

	all = append(all, AnthropicEventToResponsesEvents(&AnthropicStreamEvent{
		Type: "content_block_start",
		ContentBlock: &AnthropicContentBlock{
			Type:  "tool_use",
			ID:    "toolu_01abc",
			Name:  "eval",
			Input: blockInput,
		},
	}, state)...)

	for _, partial := range partials {
		all = append(all, AnthropicEventToResponsesEvents(&AnthropicStreamEvent{
			Type:  "content_block_delta",
			Delta: &AnthropicDelta{Type: "input_json_delta", PartialJSON: partial},
		}, state)...)
	}

	all = append(all, AnthropicEventToResponsesEvents(&AnthropicStreamEvent{Type: "content_block_stop"}, state)...)
	all = append(all, AnthropicEventToResponsesEvents(&AnthropicStreamEvent{
		Type:  "message_delta",
		Usage: &AnthropicUsage{OutputTokens: 9},
	}, state)...)
	all = append(all, AnthropicEventToResponsesEvents(&AnthropicStreamEvent{Type: "message_stop"}, state)...)

	return all
}

func concatArgumentDeltas(events []ResponsesStreamEvent) string {
	out := ""
	for _, e := range events {
		if e.Type == "response.function_call_arguments.delta" {
			out += e.Delta
		}
	}
	return out
}

func findFunctionCallOutput(events []ResponsesStreamEvent) *ResponsesOutput {
	for i := range events {
		if events[i].Type != "response.output_item.done" || events[i].Item == nil {
			continue
		}
		if events[i].Item.Type == "function_call" {
			return events[i].Item
		}
	}
	return nil
}

// 部分兼容上游在 start 帧直接给完整工具参数，不再给 delta；
// 流式链路必须保留这些参数，否则客户端只会收到空对象而无法执行工具。
func TestAnthropicEventToResponses_ToolInputOnContentBlockStart(t *testing.T) {
	const args = `{"language":"py","code":"print(1)"}`

	events := collectToolCallStreamEvents(t, json.RawMessage(args), nil)

	item := findFunctionCallOutput(events)
	require.NotNil(t, item, "a function_call output item must be emitted")
	assert.Equal(t, args, item.Arguments,
		"arguments carried on content_block_start must survive to output_item.done")
	assert.Equal(t, "eval", item.Name)

	// 按事件累加参数的客户端同样必须收到参数。
	assert.Equal(t, args, concatArgumentDeltas(events),
		"argument deltas must reconstruct the full arguments JSON")

	done := findEvent(events, "response.function_call_arguments.done")
	require.NotNil(t, done, "function_call_arguments.done must be emitted")
	assert.Equal(t, args, done.Arguments,
		"function_call_arguments.done must carry the complete arguments")

	completed := findEvent(events, "response.completed")
	require.NotNil(t, completed)
	require.NotNil(t, completed.Response)
	require.Len(t, completed.Response.Output, 1)
	assert.Equal(t, args, completed.Response.Output[0].Arguments,
		"response.completed must carry the same arguments")
}

// 官方空 start 加参数 delta 的顺序必须保持不变。
func TestAnthropicEventToResponses_ToolInputFromDeltasUnchanged(t *testing.T) {
	const args = `{"language":"py","code":"print(1)"}`

	events := collectToolCallStreamEvents(t, json.RawMessage(`{}`),
		[]string{`{"language":"py",`, `"code":"print(1)"}`})

	item := findFunctionCallOutput(events)
	require.NotNil(t, item)
	assert.Equal(t, args, item.Arguments)

	// 只保留上游的两段 delta，不重复插入 start 中的参数。
	var deltas []string
	for _, e := range events {
		if e.Type == "response.function_call_arguments.delta" {
			deltas = append(deltas, e.Delta)
		}
	}
	assert.Equal(t, []string{`{"language":"py",`, `"code":"print(1)"}`}, deltas,
		"canonical delta streaming must not gain or lose events")
}

// 同时有 start 参数与 delta 时，不得拼接成两个 JSON 文档。
func TestAnthropicEventToResponses_ToolInputSeedNotDuplicatedByDeltas(t *testing.T) {
	const args = `{"language":"py","code":"print(1)"}`

	events := collectToolCallStreamEvents(t, json.RawMessage(args),
		[]string{`{"language":"py",`, `"code":"print(1)"}`})

	item := findFunctionCallOutput(events)
	require.NotNil(t, item)
	assert.Equal(t, args, item.Arguments, "deltas win over the content_block_start seed")
	assert.True(t, json.Valid([]byte(item.Arguments)), "arguments must stay valid JSON")
	assert.Equal(t, args, concatArgumentDeltas(events), "seed must not be replayed alongside deltas")
}

// 空 start 且无 delta 仍使用原 {} 回退，不生成无意义的参数分片。
func TestAnthropicEventToResponses_ToolInputEmptyStaysEmptyObject(t *testing.T) {
	for name, input := range map[string]json.RawMessage{
		"absent": nil,
		"empty":  json.RawMessage(`{}`),
	} {
		t.Run(name, func(t *testing.T) {
			events := collectToolCallStreamEvents(t, input, nil)

			item := findFunctionCallOutput(events)
			require.NotNil(t, item)
			assert.Equal(t, "{}", item.Arguments)
			assert.Empty(t, concatArgumentDeltas(events),
				"no argument delta should be synthesized when there are no arguments")
		})
	}
}
