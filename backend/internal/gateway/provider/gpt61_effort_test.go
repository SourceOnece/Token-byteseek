package provider

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/protocol/anthropic"
	"github.com/stretchr/testify/require"
)

// 显式档位原样保留，不再从客户端模型后缀派生或偷偷升档。
func TestGPT61ExplicitEffort(t *testing.T) {
	for _, effort := range []string{"low", "medium", "high", "xhigh", "max"} {
		req := &anthropic.AnthropicRequest{Model: "public-alias", OutputConfig: &anthropic.AnthropicOutputConfig{Effort: effort}}
		require.Equal(t, effort, OpenAICompatAnthropicReasoningEffort(req, "gpt-6.1-sol", effort))
	}
}
