//go:build unit

package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 凭据交换失败后用于条件写入的快照必须保持交换前的嵌套值，不能被执行器改写。
func TestRefreshAttemptSnapshotIsolation(t *testing.T) {
	credentials := map[string]any{"refresh_token": "initial", "session": map[string]any{"cookie": "initial"}}
	provider := &Record{ID: 1, Platform: PlatformOpenAI, Type: ProviderTypeOAuth, Credentials: credentials}
	attempted := snapshotRefreshRecord(provider)
	session, ok := credentials["session"].(map[string]any)
	require.True(t, ok)
	session["cookie"] = "changed-during-exchange"
	expected, ok := attempted.Credentials["session"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "initial", expected["cookie"])
}
