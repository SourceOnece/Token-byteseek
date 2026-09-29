package transfer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// 导入必须显式使用新版格式，旧集合即使与新集合同时出现也不能忽略。
func TestProviderArchiveVersionBoundary(t *testing.T) {
	var payload DataPayload
	require.NoError(t, json.Unmarshal([]byte(`{"type":"sub2api-data","version":2,"proxies":[],"providers":[]}`), &payload))
	require.NoError(t, ValidateHeader(payload))
	for _, version := range []int{0, 1, 3} {
		candidate := payload
		candidate.Version = version
		require.Error(t, ValidateHeader(candidate))
	}
	for _, format := range []string{"", "sub2api-bundle"} {
		candidate := payload
		candidate.Type = format
		require.Error(t, ValidateHeader(candidate))
	}
	require.Error(t, json.Unmarshal([]byte(`{"type":"sub2api-data","version":2,"proxies":[],"providers":[],"accounts":[]}`), &payload))
}
