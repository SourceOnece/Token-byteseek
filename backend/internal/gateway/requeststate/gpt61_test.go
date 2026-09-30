package requeststate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 覆盖三类入站及 WS 载荷，显式关闭推理不能静默升档。
func TestGPT61RejectsDisabledReasoningAcrossShapes(t *testing.T) {
	for _, body := range []string{`{"reasoning":{"effort":"none"}}`, `{"reasoning_effort":"minimal"}`, `{"output_config":{"effort":"none"}}`, `{"thinking":{"type":"disabled"}}`, `{"response":{"reasoning":{"effort":"none"}}}`, `{"model":"gpt-6.1-sol-none"}`} {
		require.Error(t, ValidateOpenAIReasoningEffort([]byte(body), "gpt-6.1-sol"), body)
		require.NoError(t, ValidateOpenAIReasoningEffort([]byte(body), "gpt-5.4"), body)
	}
	for _, effort := range []string{"low", "medium", "high", "xhigh", "max"} {
		require.NoError(t, ValidateOpenAIReasoningEffort([]byte(`{"reasoning":{"effort":"`+effort+`"}}`), "openai/gpt-6.1-sol"))
	}
}
