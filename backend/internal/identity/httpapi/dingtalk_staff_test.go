package httpapi

import (
	"context"
	"errors"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/identity"
	"github.com/stretchr/testify/require"
)

// dingTalkStaffClient 只替换外部资料查询，策略执行使用回调的实际入口。
type dingTalkStaffClient struct {
	identity.DingTalkOAuthClient
	userErr, staffErr error
	calls             []string
	profile           *identity.DingTalkProfileSnapshot
}

func (c *dingTalkStaffClient) GetUserIdByUnionId(_ context.Context, id string) (string, error) {
	c.calls = append(c.calls, "user:"+id)
	return "staff-id", c.userErr
}

func (c *dingTalkStaffClient) GetStaffInfoByUserId(_ context.Context, id string) (*identity.DingTalkProfileSnapshot, error) {
	c.calls = append(c.calls, "staff:"+id)
	return c.profile, c.staffErr
}

// TestDingTalkStaffLookupPolicy 验证实际回调使用的失败边界、查询顺序和成功资料。
func TestDingTalkStaffLookupPolicy(t *testing.T) {
	for _, policy := range []string{"none", "", "internal_only", "unknown"} {
		for _, failure := range []string{"", "get_user_id", "get_staff_info"} {
			t.Run(policy+"/"+failure, func(t *testing.T) {
				upstreamErr := errors.New("directory unavailable")
				client := &dingTalkStaffClient{profile: &identity.DingTalkProfileSnapshot{Nickname: "tester"}}
				if failure == "get_user_id" {
					client.userErr = upstreamErr
				}
				if failure == "get_staff_info" {
					client.staffErr = upstreamErr
				}
				profile, step, err := loadDingTalkStaff(t.Context(), client, policy, "union-id", "corp-id")
				if failure == "get_user_id" {
					require.Equal(t, []string{"user:union-id"}, client.calls)
				} else {
					require.Equal(t, []string{"user:union-id", "staff:staff-id"}, client.calls)
				}
				if policy == "internal_only" && failure != "" {
					require.ErrorIs(t, err, upstreamErr)
					require.Equal(t, failure, step)
					require.Nil(t, profile)
				} else {
					require.NoError(t, err)
					require.Empty(t, step)
					if failure == "" {
						require.Same(t, client.profile, profile)
					} else {
						require.Equal(t, &identity.DingTalkProfileSnapshot{}, profile)
					}
				}
			})
		}
	}
}
