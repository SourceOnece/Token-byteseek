package quality

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 原有完整回答门禁在原生执行器接入后仍生效，不能把推理内容或半截流判成满血。
func TestQualityVisibleCompletedAnswer(t *testing.T) {
	for _, tc := range []struct {
		name, wire, want string
		invalid          bool
	}{
		{"completed", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"PASS\"}\n\ndata: {\"type\":\"response.completed\"}\n\n", "PASS", false},
		{"truncated", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"PASS\"}\n\n", "", true},
		{"failed", "data: {\"type\":\"response.failed\"}\n\n", "", true},
		{"reasoning", "data: {\"type\":\"response.reasoning_summary_text.delta\",\"delta\":\"PASS\"}\n\ndata: {\"type\":\"response.completed\"}\n\n", "", false},
		{"done_only", "data: [DONE]\n\n", "", true},
		{"incomplete", "data: {\"type\":\"response.completed\",\"response\":{\"status\":\"incomplete\"}}\n\n", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			answer, err := Responses(strings.NewReader(tc.wire))
			if tc.invalid {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.want, answer)
		})
	}
}

func TestQualityTimeoutAndProtocolContract(t *testing.T) {
	r := CodexQualityRequest{AccountIDs: []int64{1}, Model: "gpt-6-astra", Prompt: "test", Keyword: "PASS", ConfirmScheduling: true}
	require.NoError(t, r.Normalize())
	require.Equal(t, 120, r.TimeoutSeconds)
	r.TimeoutSeconds = 300
	r.APIProtocol = "chat_completions"
	require.NoError(t, r.Normalize())
	r.APIProtocol = "all"
	require.Error(t, r.Normalize())
	r.APIProtocol = "responses"
	r.TimeoutSeconds = 3601
	require.Error(t, r.Normalize())
}
