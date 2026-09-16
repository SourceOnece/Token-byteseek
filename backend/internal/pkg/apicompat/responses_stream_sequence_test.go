package apicompat

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// grok-build 把 sequence_number 当必填。response.created 从 0 起号，
// omitempty 会把 0 整段丢掉，第一帧就反序列化失败。
func TestWire_SequenceNumberPresentAtZero(t *testing.T) {
	created := marshalEvent(t, ResponsesStreamEvent{
		Type:     "response.created",
		Response: &ResponsesResponse{ID: "resp_1", Object: "response", Status: "in_progress"},
	})
	require.Contains(t, created, "sequence_number")
	require.EqualValues(t, 0, created["sequence_number"])

	completed := marshalEvent(t, ResponsesStreamEvent{
		Type:           "response.completed",
		SequenceNumber: 0,
		Response:       &ResponsesResponse{ID: "resp_1", Object: "response", Status: "completed"},
	})
	require.Contains(t, completed, "sequence_number")
	require.EqualValues(t, 0, completed["sequence_number"])

	delta := marshalEvent(t, ResponsesStreamEvent{
		Type: "response.output_text.delta", OutputIndex: 0, ContentIndex: 0, ItemID: "msg_1", Delta: "hi",
	})
	require.Contains(t, delta, "sequence_number")
	require.EqualValues(t, 0, delta["sequence_number"])
}
