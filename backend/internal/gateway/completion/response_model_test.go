package completion

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResponseModelMismatchUsesOutboundIdentity 验证别名与价格模型不会参与观测比较。
func TestResponseModelMismatchUsesOutboundIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, sent, response string
		known, want          bool
	}{
		{"same", "runtime", "runtime", true, false},
		{"case and whitespace", " Runtime ", "runtime", true, false},
		{"version", "runtime", "runtime-20260101", true, true},
		{"prefix", "vendor/runtime", "runtime", true, true},
		{"missing", "runtime", "", false, false},
		{"no mapping", "", "client", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &Result{Model: "client", BillingModel: "price", UpstreamModel: tc.sent, UpstreamResponseModel: tc.response}
			got := responseModelMismatch(r)
			if !tc.known {
				require.Nil(t, got)
			} else {
				require.NotNil(t, got)
				require.Equal(t, tc.want, *got)
			}
			require.Equal(t, "price", r.BillingModel)
		})
	}
}

// TestResponseModelCompletionSnapshot 验证排队后的记录不读取后续 turn 的模型。
func TestResponseModelCompletionSnapshot(t *testing.T) {
	source := &Input{Result: &Result{UpstreamModel: "sent", UpstreamResponseModel: "first"}}
	captured := Snapshot(source)
	source.Result.UpstreamResponseModel = "next-turn"
	require.Equal(t, "first", captured.Result.UpstreamResponseModel)
}

// 旧字段兼容历史定制，新观测存在时拥有优先级，不能混用路由或计费模型。
func TestResponseModelLegacyCompatibility(t *testing.T) {
	r := &Result{Model: "request", UpstreamModel: "sent", ResponseModel: "legacy"}
	require.Equal(t, "legacy", recordedResponseModel(r))
	require.True(t, *responseModelMismatch(r))
	r.UpstreamResponseModel = "sent"
	require.Equal(t, "sent", recordedResponseModel(r))
	require.False(t, *responseModelMismatch(r))
}
