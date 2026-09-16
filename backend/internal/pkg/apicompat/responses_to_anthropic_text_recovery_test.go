package apicompat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 使用同一转换状态消费一组上游事件。
func feedResponsesEvents(events ...*ResponsesStreamEvent) []AnthropicStreamEvent {
	state := NewResponsesEventToAnthropicState()
	var out []AnthropicStreamEvent
	for _, evt := range events {
		out = append(out, ResponsesEventToAnthropicEvents(evt, state)...)
	}
	return out
}

// 汇总转换后 Claude 客户端实际收到的可见文本。
func collectAnthropicText(events []AnthropicStreamEvent) string {
	text := ""
	for _, event := range events {
		if event.Type == "content_block_delta" && event.Delta != nil && event.Delta.Type == "text_delta" {
			text += event.Delta.Text
		}
	}
	return text
}

// 严格验证块生命周期：索引递增、块成对关闭且增量不出现在块外。
func requireAnthropicBlockLifecycle(t *testing.T, events []AnthropicStreamEvent) {
	t.Helper()

	open := false
	next := 0
	for _, event := range events {
		switch event.Type {
		case "content_block_start":
			require.False(t, open, "content_block_start while a block is open")
			require.NotNil(t, event.Index)
			require.Equal(t, next, *event.Index, "content block indices must increase by one")
			open = true
		case "content_block_delta":
			require.True(t, open, "content_block_delta outside an open block")
			require.NotNil(t, event.Index)
			require.Equal(t, next, *event.Index)
		case "content_block_stop":
			require.True(t, open, "content_block_stop without an open block")
			require.NotNil(t, event.Index)
			require.Equal(t, next, *event.Index)
			open = false
			next++
		}
	}
	require.False(t, open, "stream ended with an unclosed content block")
}

func responsesCreated() *ResponsesStreamEvent {
	return &ResponsesStreamEvent{
		Type:     "response.created",
		Response: &ResponsesResponse{ID: "resp_text_recovery", Model: "gpt-5.2"},
	}
}

func responsesMessageOutput(texts ...string) []ResponsesOutput {
	content := make([]ResponsesContentPart, 0, len(texts))
	for _, text := range texts {
		content = append(content, ResponsesContentPart{Type: "output_text", Text: text})
	}
	return []ResponsesOutput{{Type: "message", Role: "assistant", Content: content}}
}

// 只有 output_text.done 而没有 delta 时，也要交付完整文本。
func TestResponsesEventToAnthropicEvents_RecoversTextFromDoneEvent(t *testing.T) {
	const answer = "recovered from done"

	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.output_item.added", Item: &ResponsesOutput{Type: "message"}},
		&ResponsesStreamEvent{Type: "response.output_text.done", Text: answer},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{Status: "completed"}},
	)

	assert.Equal(t, answer, collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
	require.NotEmpty(t, events)
	assert.Equal(t, "message_stop", events[len(events)-1].Type)
}

// 终态独有的文本必须补发，避免空回答。
func TestResponsesEventToAnthropicEvents_RecoversTextFromTerminalOutput(t *testing.T) {
	const answer = "recovered from terminal"

	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{
			Status: "completed",
			Output: responsesMessageOutput(answer),
		}},
	)

	assert.Equal(t, answer, collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
	require.NotEmpty(t, events)
	assert.Equal(t, "message_stop", events[len(events)-1].Type)
}

// 正常增量已交付的文本不得被 done 或终态重复发送。
func TestResponsesEventToAnthropicEvents_DoesNotDuplicateTextDeliveredByDeltas(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.output_text.delta", Delta: "Hel"},
		&ResponsesStreamEvent{Type: "response.output_text.delta", Delta: "lo"},
		&ResponsesStreamEvent{Type: "response.output_text.done", Text: "Hello"},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{
			Status: "completed",
			Output: responsesMessageOutput("Hello"),
		}},
	)

	assert.Equal(t, "Hello", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}

// done 补发过的文本不得再由终态重复。
func TestResponsesEventToAnthropicEvents_DoesNotDuplicateTextRecoveredFromDoneEvent(t *testing.T) {
	const answer = "recovered once"

	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.output_text.done", Text: answer},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{
			Status: "completed",
			Output: responsesMessageOutput(answer),
		}},
	)

	assert.Equal(t, answer, collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}

// 去重按输出分片记录，内容块关闭不能清空交付历史。
func TestResponsesEventToAnthropicEvents_DoesNotDuplicateTextWhenBlockClosedBeforeDone(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.output_text.delta", Delta: "Hello"},
		&ResponsesStreamEvent{
			Type:        "response.output_item.added",
			OutputIndex: 1,
			Item:        &ResponsesOutput{Type: "function_call", CallID: "call_1", Name: "lookup"},
		},
		&ResponsesStreamEvent{Type: "response.output_text.done", Text: "Hello"},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{Status: "completed"}},
	)

	assert.Equal(t, "Hello", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}

// 部分增量后收到完整 done 时，只补缺少的后缀。
func TestResponsesEventToAnthropicEvents_AppendsOnlySuffixMissingFromDeltas(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.output_text.delta", Delta: "Hel"},
		&ResponsesStreamEvent{Type: "response.output_text.done", Text: "Hello world"},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{
			Status: "completed",
			Output: responsesMessageOutput("Hello world"),
		}},
	)

	assert.Equal(t, "Hello world", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}

// 流式文本不可撤回，冲突的终态不追加到已发送内容。
func TestResponsesEventToAnthropicEvents_IgnoresDonePayloadDivergingFromDeltas(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.output_text.delta", Delta: "Hello"},
		&ResponsesStreamEvent{Type: "response.output_text.done", Text: "Goodbye"},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{Status: "completed"}},
	)

	assert.Equal(t, "Hello", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}

// 事件索引与终态数组不保证相同；已有文本时不猜测映射。
func TestResponsesEventToAnthropicEvents_DoesNotDuplicateWhenTerminalOutputIsIndexedDifferently(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{
			Type:        "response.output_item.added",
			OutputIndex: 0,
			Item:        &ResponsesOutput{Type: "reasoning", ID: "rs_1"},
		},
		&ResponsesStreamEvent{Type: "response.output_text.delta", OutputIndex: 1, Delta: "Hello"},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{
			Status: "completed",
			// 此处终态省略了 reasoning，数组位置与增量的输出索引不同。
			Output: responsesMessageOutput("Hello"),
		}},
	)

	assert.Equal(t, "Hello", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}

// 部分增量后不能依靠终态数组的不同索引重复补发。
func TestResponsesEventToAnthropicEvents_LeavesTerminalOutputAloneOnceTextStreamed(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.output_text.delta", ContentIndex: 0, Delta: "first"},
		&ResponsesStreamEvent{Type: "response.output_text.done", ContentIndex: 0, Text: "first"},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{
			Status: "completed",
			Output: responsesMessageOutput("first", "second"),
		}},
	)

	assert.Equal(t, "first", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}

// 完全没有文本增量时，恢复终态中的所有文本分片。
func TestResponsesEventToAnthropicEvents_RecoversEveryTerminalPartWhenNothingStreamed(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{
			Status: "completed",
			Output: responsesMessageOutput("first", "second"),
		}},
	)

	assert.Equal(t, "firstsecond", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)

	starts := 0
	for _, event := range events {
		if event.Type == "content_block_start" {
			starts++
		}
	}
	assert.Equal(t, 1, starts, "contiguous recovered text belongs in one block")
}

// 只有工具调用的回合不能凭空产生文本。
func TestResponsesEventToAnthropicEvents_LeavesToolOnlyTurnTextFree(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{
			Type:        "response.output_item.added",
			OutputIndex: 0,
			Item:        &ResponsesOutput{Type: "function_call", CallID: "call_1", Name: "lookup"},
		},
		&ResponsesStreamEvent{
			Type:        "response.function_call_arguments.done",
			OutputIndex: 0,
			Arguments:   `{"id":1}`,
		},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{
			Status: "completed",
			Output: []ResponsesOutput{{Type: "function_call", CallID: "call_1", Name: "lookup", Arguments: `{"id":1}`}},
		}},
	)

	assert.Empty(t, collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}

// 所有受支持的终止事件别名保持相同恢复行为。
func TestResponsesEventToAnthropicEvents_RecoversTextOnEveryTerminalAlias(t *testing.T) {
	for _, terminal := range []string{
		"response.completed",
		"response.done",
		"response.incomplete",
		"response.failed",
	} {
		t.Run(terminal, func(t *testing.T) {
			const answer = "recovered on a terminal alias"

			events := feedResponsesEvents(
				responsesCreated(),
				&ResponsesStreamEvent{Type: terminal, Response: &ResponsesResponse{
					Status: "completed",
					Output: responsesMessageOutput(answer),
				}},
			)

			assert.Equal(t, answer, collectAnthropicText(events))
			requireAnthropicBlockLifecycle(t, events)
			require.NotEmpty(t, events)
			assert.Equal(t, "message_stop", events[len(events)-1].Type)
		})
	}
}

// 无法与增量分片对应的 done 必须忽略，避免重复回答。
func TestResponsesEventToAnthropicEvents_DoesNotDuplicateWhenDoneIsIndexedDifferently(t *testing.T) {
	for _, tc := range []struct {
		name string
		done *ResponsesStreamEvent
	}{
		{"content index diverges", &ResponsesStreamEvent{Type: "response.output_text.done", ContentIndex: 1, Text: "Hello world"}},
		{"output index diverges", &ResponsesStreamEvent{Type: "response.output_text.done", OutputIndex: 1, Text: "Hello world"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			events := feedResponsesEvents(
				responsesCreated(),
				&ResponsesStreamEvent{Type: "response.output_text.delta", Delta: "Hello world"},
				tc.done,
				&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{Status: "completed"}},
			)

			assert.Equal(t, "Hello world", collectAnthropicText(events))
			requireAnthropicBlockLifecycle(t, events)
		})
	}
}

// 部分增量也适用未知分片不补发的规则。
func TestResponsesEventToAnthropicEvents_LeavesUnmatchableDoneAloneAfterPartialDeltas(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.output_text.delta", ContentIndex: 0, Delta: "Hello"},
		&ResponsesStreamEvent{Type: "response.output_text.done", ContentIndex: 1, Text: "Hello world"},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{Status: "completed"}},
	)

	assert.Equal(t, "Hello", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}

// 同时使用 output_index 和 content_index 区分文本分片。
func TestResponsesEventToAnthropicEvents_KeysDeliveredTextByBothIndices(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.output_text.delta", OutputIndex: 0, ContentIndex: 0, Delta: "first"},
		&ResponsesStreamEvent{Type: "response.output_text.done", OutputIndex: 0, ContentIndex: 0, Text: "first"},
		&ResponsesStreamEvent{Type: "response.output_text.delta", OutputIndex: 0, ContentIndex: 1, Delta: "second"},
		&ResponsesStreamEvent{Type: "response.output_text.done", OutputIndex: 0, ContentIndex: 1, Text: "second-tail"},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{Status: "completed"}},
	)

	assert.Equal(t, "firstsecond-tail", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}

// message_stop 后的迟到 done 不得重新开启内容块。
func TestResponsesEventToAnthropicEvents_IgnoresDoneEventAfterMessageStop(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{
			Status: "completed",
			Output: responsesMessageOutput("the answer"),
		}},
		&ResponsesStreamEvent{Type: "response.output_text.done", OutputIndex: 7, Text: "late text"},
	)

	assert.Equal(t, "the answer", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
	require.NotEmpty(t, events)
	assert.Equal(t, "message_stop", events[len(events)-1].Type, "no content may follow message_stop")
}

// 工具完成事件只处理参数，不把额外 text 当成回答。
func TestResponsesEventToAnthropicEvents_LeavesToolArgumentsDoneTextFree(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{
			Type:        "response.output_item.added",
			OutputIndex: 0,
			Item:        &ResponsesOutput{Type: "function_call", CallID: "call_1", Name: "lookup"},
		},
		&ResponsesStreamEvent{
			Type:        "response.function_call_arguments.done",
			OutputIndex: 0,
			Arguments:   `{"id":1}`,
			Text:        "must not be emitted",
		},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{Status: "completed"}},
	)

	assert.Empty(t, collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}

// 恢复文本先走公共闭块逻辑，保留 thinking 签名。
func TestResponsesEventToAnthropicEvents_PreservesThinkingSignatureWhenRecovering(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{
			Type:        "response.output_item.added",
			OutputIndex: 0,
			Item:        &ResponsesOutput{Type: "reasoning", EncryptedContent: "sig-abc"},
		},
		&ResponsesStreamEvent{Type: "response.output_text.done", OutputIndex: 1, Text: "recovered"},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{Status: "completed"}},
	)

	assert.Equal(t, "recovered", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)

	var signatures, stopsBefore int
	for _, event := range events {
		if event.Type == "content_block_delta" && event.Delta != nil && event.Delta.Type == "signature_delta" {
			assert.Equal(t, "sig-abc", event.Delta.Signature)
			signatures++
			assert.Equal(t, 0, stopsBefore, "signature_delta must precede the thinking block stop")
		}
		if event.Type == "content_block_stop" {
			stopsBefore++
		}
	}
	assert.Equal(t, 1, signatures, "the thinking signature must survive recovery")
}

// 缺少正式终态时，合成结束逻辑仍须关闭恢复文本的块。
func TestResponsesEventToAnthropicEvents_RecoveredTextSurvivesTheSyntheticFinalizer(t *testing.T) {
	state := NewResponsesEventToAnthropicState()
	var events []AnthropicStreamEvent
	for _, evt := range []*ResponsesStreamEvent{
		responsesCreated(),
		{Type: "response.output_text.done", Text: "recovered before the stream died"},
	} {
		events = append(events, ResponsesEventToAnthropicEvents(evt, state)...)
	}
	events = append(events, FinalizeResponsesAnthropicStream(state)...)

	assert.Equal(t, "recovered before the stream died", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
	require.NotEmpty(t, events)
	assert.Equal(t, "message_stop", events[len(events)-1].Type)
}

// 终态仅恢复 message 项中的 output_text。
func TestResponsesEventToAnthropicEvents_DeclinesTerminalRecoveryWithoutMessageText(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response *ResponsesResponse
	}{
		{"no response payload", nil},
		{"no message item", &ResponsesResponse{Status: "completed", Output: []ResponsesOutput{{Type: "reasoning"}}}},
		{"message without output_text", &ResponsesResponse{Status: "completed", Output: []ResponsesOutput{{
			Type:    "message",
			Role:    "assistant",
			Content: []ResponsesContentPart{{Type: "refusal", Text: "refused"}},
		}}}},
		{"empty output_text", &ResponsesResponse{Status: "completed", Output: responsesMessageOutput("")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			events := feedResponsesEvents(
				responsesCreated(),
				&ResponsesStreamEvent{Type: "response.completed", Response: tc.response},
			)

			assert.Empty(t, collectAnthropicText(events))
			requireAnthropicBlockLifecycle(t, events)
		})
	}
}

// 恢复全部 message 文本，跳过中间的非文本输出项。
func TestResponsesEventToAnthropicEvents_RecoversEveryTerminalMessageItem(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{
			Status: "completed",
			Output: []ResponsesOutput{
				{Type: "reasoning"},
				{Type: "message", Role: "assistant", Content: []ResponsesContentPart{{Type: "output_text", Text: "first"}}},
				{Type: "function_call", CallID: "call_1", Name: "lookup"},
				{Type: "message", Role: "assistant", Content: []ResponsesContentPart{{Type: "output_text", Text: "second"}}},
			},
		}},
	)

	assert.Equal(t, "firstsecond", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}

// 工具块打开时收到文本 done，应先关闭工具块再补文本。
func TestResponsesEventToAnthropicEvents_EmitsDoneTextWhileAToolBlockIsOpen(t *testing.T) {
	events := feedResponsesEvents(
		responsesCreated(),
		&ResponsesStreamEvent{
			Type:        "response.output_item.added",
			OutputIndex: 0,
			Item:        &ResponsesOutput{Type: "function_call", CallID: "call_1", Name: "lookup"},
		},
		&ResponsesStreamEvent{Type: "response.output_text.done", OutputIndex: 1, Text: "spoken after the call"},
		&ResponsesStreamEvent{Type: "response.completed", Response: &ResponsesResponse{Status: "completed"}},
	)

	assert.Equal(t, "spoken after the call", collectAnthropicText(events))
	requireAnthropicBlockLifecycle(t, events)
}
