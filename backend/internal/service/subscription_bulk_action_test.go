package service

import (
	"context"
	infraerrors "github.com/TokenFlux/TokenRouter/internal/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBulkSubscriptionAction_ValidatesBeforeAnyRepositoryAccess(t *testing.T) {
	tooMany := make([]int64, MaxBulkSubscriptionActions+1)
	for i := range tooMany {
		tooMany[i] = 1
	}
	for name, input := range map[string]*BulkSubscriptionActionInput{
		"nil":              nil,
		"empty IDs":        {Action: "revoke"},
		"too many IDs":     {SubscriptionIDs: tooMany, Action: "revoke"},
		"invalid later ID": {SubscriptionIDs: []int64{1, 0}, Action: "revoke"},
		"negative ID":      {SubscriptionIDs: []int64{1, -1}, Action: "revoke"},
		"unknown action":   {SubscriptionIDs: []int64{1}, Action: "delete"},
		"missing action":   {SubscriptionIDs: []int64{1}},
		"zero adjustment":  {SubscriptionIDs: []int64{1}, Action: "extend"},
		"large adjustment": {SubscriptionIDs: []int64{1}, Action: "extend", Days: MaxValidityDays + 1},
		"small adjustment": {SubscriptionIDs: []int64{1}, Action: "extend", Days: -MaxValidityDays - 1},
		"no reset windows": {SubscriptionIDs: []int64{1}, Action: "reset_quota"},
	} {
		t.Run(name, func(t *testing.T) {
			// 空仓储确保整批校验失败前没有产生读取或写入。
			svc := &SubscriptionService{}
			result, err := svc.BulkSubscriptionAction(context.Background(), input)
			require.Error(t, err)
			require.Equal(t, 400, infraerrors.Code(err))
			require.Nil(t, result)
		})
	}
	for _, days := range []int{-MaxValidityDays, -1, 1, MaxValidityDays} {
		input := BulkSubscriptionActionInput{SubscriptionIDs: []int64{1}, Action: "extend", Days: days}
		require.NoError(t, input.Validate())
	}
}
