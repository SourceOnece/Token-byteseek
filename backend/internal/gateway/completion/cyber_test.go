package completion

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// Cyber 只改变记录分类，继续复用原计费与失败事实路径。
func TestRecordCyberRetainsFailureFactAndDoesNotResettleLogFailure(t *testing.T) {
	for _, failed := range []bool{false, true} {
		core, funds, logs, in, _ := recordFixture()
		if failed {
			funds.err = errors.New("settlement failed")
		} else {
			logs.bestErr = errors.New("queue full")
			logs.errorSync = errors.New("write failed")
		}
		core.RecordCyber(context.Background(), in)
		require.Equal(t, 1, funds.calls)
		require.NotEmpty(t, logs.rows)
		for _, row := range logs.rows {
			require.Equal(t, RequestTypeCyberBlocked, row.RequestType)
			if failed {
				require.Zero(t, row.ActualCost)
			}
		}
		require.False(t, in.CyberBlocked)
	}
}

// 零用量审核记录也进入结算，金额保持为零。
func TestRecordCyberPreservesZeroUsage(t *testing.T) {
	core, funds, logs, in, _ := recordFixture()
	in.Result.Usage = TokenUsage{}
	core.RecordCyber(context.Background(), in)
	require.Equal(t, 1, funds.calls)
	require.Zero(t, funds.command.BillableAmountUSD)
	require.NotEmpty(t, logs.rows)
	require.Zero(t, logs.rows[0].ActualCost)
	require.Equal(t, RequestTypeCyberBlocked, logs.rows[0].RequestType)
}
