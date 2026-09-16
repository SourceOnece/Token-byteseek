package apicompat

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// 终态输出项被重排后，补参必须优先使用调用 ID，不能按旧位置串用参数。
func TestBufferedResponseAccumulator_DoneUsesCallIDBeforePosition(t *testing.T) {
	a := NewBufferedResponseAccumulator()
	for i, id := range []string{"a", "b"} {
		a.ProcessEvent(&ResponsesStreamEvent{Type: "response.output_item.added", OutputIndex: i, Item: &ResponsesOutput{Type: "function_call", CallID: id, Name: "exec"}})
		a.ProcessEvent(&ResponsesStreamEvent{Type: "response.function_call_arguments.done", OutputIndex: i, Arguments: id})
	}
	r := &ResponsesResponse{Output: []ResponsesOutput{{Type: "function_call", CallID: "b"}, {Type: "function_call", CallID: "a"}}}
	a.SupplementResponseOutput(r)
	require.Equal(t, "b", r.Output[0].Arguments)
	require.Equal(t, "a", r.Output[1].Arguments)
}

// 工具完成只读对应协议字段，终态之后的迟到事件不应新增 delta。
func TestResponsesChatToolDoneFinalizedIsNoop(t *testing.T) {
	s := NewResponsesEventToChatState()
	s.OutputIndexToToolIndex[0] = 0
	s.Finalized = true
	require.Empty(t, ResponsesEventToChatChunks(&ResponsesStreamEvent{Type: "response.function_call_arguments.done", Arguments: "{}"}, s))
}
