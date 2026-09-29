package httpapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Messages 不再增加协议专用映射层，保留通用分组映射结果和既有协议型号规范化。
func TestMessagesProviderModelKeepsGroupMapping(t *testing.T) {
	require.Equal(t, "group-model", ResolveOpenAIMessagesProviderLayerModel("group-model"))
	require.Equal(t, "claude-sonnet-4-6", ResolveOpenAIMessagesProviderLayerModel("claude-sonnet-4-6"))
	require.Equal(t, "gpt-5.4", ResolveOpenAIMessagesProviderLayerModel("gpt-5.4-high"))
}
