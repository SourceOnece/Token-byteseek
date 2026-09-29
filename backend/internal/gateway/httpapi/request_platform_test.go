package httpapi

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/apikey"

	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/stretchr/testify/require"
)

// 分组不再决定上游，未选号时保留空平台以允许跨平台候选。
func TestOpenAICompatibleRequestPlatformStaysUnspecifiedBeforeSelection(t *testing.T) {
	require.Empty(t, OpenAICompatibleRequestPlatform(nil))
	require.Empty(t, OpenAICompatibleRequestPlatform(&apikey.APIKey{Group: &routing.Group{}}))
}
