package apikey

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/stretchr/testify/require"
)

// 真实创建用例验证自定义/随机 Key 共用窗口。
type frequencyRepo struct {
	rotationRepository
	created int
}

func (r *frequencyRepo) Create(_ context.Context, key *APIKey) error {
	r.created++
	key.ID = int64(r.created)
	r.key = key
	return nil
}
func (r *frequencyRepo) ExistsByKey(context.Context, string) (bool, error) { return false, nil }

type frequencyUser struct{ UserRepository }

func (frequencyUser) GetByID(context.Context, int64) (*User, error) {
	return &User{ID: 7, Status: "active"}, nil
}

type frequencyGroup struct{ GroupRepository }

func (frequencyGroup) GetByID(context.Context, int64) (*routing.Group, error) {
	return &routing.Group{ID: 3, Status: "active"}, nil
}

type frequencyCache struct {
	rotationCache
	count  int64
	failed bool
}

func (c *frequencyCache) IncrementCreateCount(_ context.Context, id int64, window time.Duration) (int64, error) {
	if id != 7 || window != time.Hour {
		panic("创建计数必须按操作用户与一小时窗口")
	}
	if c.failed {
		return 0, errors.New("offline")
	}
	c.count++
	return c.count, nil
}
func (*frequencyCache) GetCreateAttemptCount(context.Context, int64) (int, error) { return 0, nil }

func TestCreateFrequencySharedByGeneratedAndCustomCredentials(t *testing.T) {
	repo := &frequencyRepo{}
	cache := &frequencyCache{rotationCache: rotationCache{entries: map[string]*APIKeyAuthCacheEntry{}}}
	options := &Options{MaxCreatesPerHour: 2}
	svc := NewAPIKeyService(repo, frequencyUser{}, frequencyGroup{}, nil, nil, cache, options)
	t.Cleanup(svc.Stop)
	group := int64(3)
	custom := "sk-custom-synthetic-test"
	for _, credential := range []*string{nil, &custom} {
		_, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{GroupID: &group, CustomKey: credential})
		require.NoError(t, err)
	}
	_, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{GroupID: &group})
	require.ErrorIs(t, err, ErrAPIKeyCreateLimited)
	require.Equal(t, 2, repo.created)
	cache.failed = true
	_, err = svc.Create(context.Background(), 7, CreateAPIKeyRequest{GroupID: &group})
	require.NoError(t, err)
	cache.failed = false
	options.MaxCreatesPerHour = 0
	_, err = svc.Create(context.Background(), 7, CreateAPIKeyRequest{GroupID: &group})
	require.NoError(t, err)
}
