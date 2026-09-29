package provider

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// 记录顺序与故障短路，不能用最终字段断言掩盖前序失败后仍解除运行阻断的问题。
type adminStateRecorder struct {
	ops     []string
	fail    string
	shadows []int64
	AdminStateStore
}

var errAdminStateTest = errors.New("synthetic storage failure")

func (s *adminStateRecorder) step(op string) error {
	s.ops = append(s.ops, op)
	if s.fail == op {
		return errAdminStateTest
	}
	return nil
}
func (s *adminStateRecorder) ListShadowIDs(context.Context, int64) ([]int64, error) {
	return s.shadows, s.step("list_shadows")
}
func (s *adminStateRecorder) Delete(_ context.Context, id int64) error {
	return s.step(fmt.Sprintf("delete_%d", id))
}
func (s *adminStateRecorder) SetSchedulable(_ context.Context, _ int64, enabled bool) error {
	return s.step(fmt.Sprintf("scheduling_%t", enabled))
}
func (s *adminStateRecorder) ClearError(context.Context, int64) error     { return s.step("error") }
func (s *adminStateRecorder) ClearRateLimit(context.Context, int64) error { return s.step("rate") }
func (s *adminStateRecorder) ClearAntigravityQuotaScopes(context.Context, int64) error {
	return s.step("quota")
}
func (s *adminStateRecorder) ClearModelRateLimits(context.Context, int64) error {
	return s.step("models")
}
func (s *adminStateRecorder) ClearTempUnschedulable(context.Context, int64) error {
	return s.step("cooldown")
}
func (s *adminStateRecorder) ClearProviderSchedulingBlock(int64) { _ = s.step("unblock") }

func TestProviderAdminClearErrorSequenceAndFailures(t *testing.T) {
	steps := []string{"error", "rate", "quota", "models", "cooldown", "unblock"}
	for i := 0; i < len(steps); i++ {
		r := &adminStateRecorder{}
		if i < len(steps)-1 {
			r.fail = steps[i]
		}
		err := NewAdmin(r, r).ClearProviderError(context.Background(), 7)
		if r.fail != "" {
			require.ErrorIs(t, err, errAdminStateTest)
			require.Equal(t, steps[:i+1], r.ops)
		} else {
			require.NoError(t, err)
			require.Equal(t, steps, r.ops)
		}
	}
}

func TestProviderAdminDeleteShadowOrderAndFailures(t *testing.T) {
	steps := []string{"list_shadows", "delete_8", "delete_9", "delete_7"}
	for i := 0; i <= len(steps); i++ {
		r := &adminStateRecorder{shadows: []int64{8, 9}}
		if i < len(steps) {
			r.fail = steps[i]
		}
		err := NewAdmin(r, nil).DeleteProvider(context.Background(), 7)
		if r.fail != "" {
			require.ErrorIs(t, err, errAdminStateTest)
			require.Equal(t, steps[:i+1], r.ops)
		} else {
			require.NoError(t, err)
			require.Equal(t, steps, r.ops)
		}
	}
}

func TestProviderAdminSchedulingDoesNotClearOtherState(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		r := &adminStateRecorder{}
		require.NoError(t, NewAdmin(r, r).SetProviderSchedulable(context.Background(), 7, enabled))
		require.Equal(t, []string{fmt.Sprintf("scheduling_%t", enabled)}, r.ops)
	}
}
