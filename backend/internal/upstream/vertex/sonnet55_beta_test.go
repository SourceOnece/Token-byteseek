package vertex

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// 验证实际 Vertex 出站 Header，不只测试共享纯函数。
func TestSonnet55ToolsetBetaOnVertexRequest(t *testing.T) {
	for _, model := range []string{"claude-sonnet-5-5", "claude-sonnet-4-6"} {
		options := AnthropicRequestOptions{
			ClientBeta:   "fine-grained-tool-streaming-2025-05-14,interleaved-thinking-2025-05-14",
			Project:      func() string { return "test-project" },
			Location:     func(string) string { return "us-east5" },
			Policy:       func(context.Context, string) (map[string]struct{}, error) { return nil, nil },
			SanitizeBody: func(body []byte, _ string) ([]byte, bool) { return body, false },
			SetHeader:    func(h http.Header, key, value string) { h.Set(key, value) },
			DeleteHeader: func(h http.Header, key string) { h.Del(key) },
			Debug:        func(http.Header, []byte, map[string]string) {},
		}
		req, err := BuildAnthropicRequest(context.Background(), []byte(`{"messages":[],"tools":[{"type":"computer_toolset_20260801"}]}`), "test-token", model, true, options)
		require.NoError(t, err)
		beta := req.Header.Get("anthropic-beta")
		require.Contains(t, beta, "interleaved-thinking")
		if model == "claude-sonnet-5-5" {
			require.NotContains(t, beta, "fine-grained-tool-streaming")
		} else {
			require.Contains(t, beta, "fine-grained-tool-streaming")
		}
	}
}
