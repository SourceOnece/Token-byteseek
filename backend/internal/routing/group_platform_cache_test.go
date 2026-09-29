//go:build unit

package routing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/stretchr/testify/require"
)

// groupPlatformRepoStub 只实现 UpdateGroup 走到的两个方法，其余靠内嵌接口占位。
type groupPlatformRepoStub struct {
	routing.GroupRepository

	group     *routing.Group
	updated   *routing.Group
	updateErr error
}

func (r *groupPlatformRepoStub) GetByID(_ context.Context, _ int64) (*routing.Group, error) {
	cloned := *r.group
	return &cloned, nil
}

func (r *groupPlatformRepoStub) Update(_ context.Context, group *routing.Group) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updated = group
	return nil
}

type pricingConfigCacheInvalidatorSpy struct {
	calls int
}

func (s *pricingConfigCacheInvalidatorSpy) InvalidateCache() { s.calls++ }

// 分组名称和策略由分组读取端口拥有，不需要重建独立的共享价格配置缓存。
func TestUpdateGroupDoesNotInvalidateIndependentPricingCache(t *testing.T) {
	repo := &groupPlatformRepoStub{group: &routing.Group{ID: 7, Name: "before"}}
	spy := &pricingConfigCacheInvalidatorSpy{}
	svc := newGroupAdminForTest(repo, nil, spy)
	got, err := svc.UpdateGroup(context.Background(), 7, &routing.UpdateGroupInput{Name: "after"})
	require.NoError(t, err)
	require.Equal(t, "after", got.Name)
	require.Zero(t, spy.calls)
}

// 依赖可以不注入（例如测试或裁剪构建），此时不应 panic——缓存靠 TTL 自然重建。
func TestUpdateGroupWithoutPricingConfigCacheInvalidator(t *testing.T) {
	repo := &groupPlatformRepoStub{group: &routing.Group{ID: 7, Name: "g"}}
	svc := newGroupAdminForTest(repo, nil, nil)

	got, err := svc.UpdateGroup(context.Background(), 7, &routing.UpdateGroupInput{})
	require.NoError(t, err)
	require.Equal(t, "g", got.Name)
}

// 分组事务失败时数据库仍保留原设置，因此不能提前失效并重建共享价格配置缓存。
func TestUpdateGroupDoesNotInvalidatePricingConfigCacheWhenUpdateFails(t *testing.T) {
	repo := &groupPlatformRepoStub{
		group:     &routing.Group{ID: 7, Name: "g"},
		updateErr: errors.New("update failed"),
	}
	spy := &pricingConfigCacheInvalidatorSpy{}
	svc := newGroupAdminForTest(repo, nil, spy)

	got, err := svc.UpdateGroup(context.Background(), 7, &routing.UpdateGroupInput{})
	require.Error(t, err)
	require.Nil(t, got)
	require.Zero(t, spy.calls)
}
