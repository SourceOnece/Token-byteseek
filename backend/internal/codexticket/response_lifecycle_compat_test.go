package codexticket

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 非标准包装只做协议等价适配，不把未知终态或错误响应视为合格。
func TestTicketCompletionLifecycleCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name, body, event string
		complete          bool
		reason            string
	}{
		{"legacy complete", `{"type":"response.done","response":{"id":"r","status":"completed","model":"gpt-6-astra"}}`, "", true, ""},
		{"legacy mismatch", `{"type":"response.done","response":{"id":"r","status":"completed","model":"gpt-5.6-sol"}}`, "", true, "model_mismatch"},
		{"missing status", `{"type":"response.done","response":{"id":"r","model":"gpt-6-astra"}}`, "", false, ""},
		{"failed", `{"type":"response.done","response":{"id":"r","status":"failed","model":"gpt-6-astra"}}`, "", false, "incomplete_response"},
		{"bare completed", `{"object":"response","id":"r","status":"completed","model":"gpt-6-astra"}`, "response.completed", true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := inspectTicketCompletion([]byte(tc.body), tc.event, "gpt-6-astra", false)
			require.Equal(t, tc.complete, got.Complete)
			require.Equal(t, tc.reason, got.Reason)
		})
	}
}
