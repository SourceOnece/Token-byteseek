package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 只实现测试所需端口，未实现的方法若被意外调用会立即失败。
type providerWriteFallbackRepo struct {
	AccountRepository
	current *Account
	readErr error
	writes  int
	written *Account
	update  func(*Account) error
}

func (r *providerWriteFallbackRepo) GetByID(context.Context, int64) (*Account, error) {
	return r.current, r.readErr
}
func (r *providerWriteFallbackRepo) Update(_ context.Context, a *Account) error {
	r.writes++
	r.written = a
	if r.update != nil {
		return r.update(a)
	}
	return nil
}

// 缺少专用凭据接口的旧实现也不能把代理、票据、分组和调度重置为零值。
func TestProviderWriteCredentialsFallbackPreservesAccount(t *testing.T) {
	proxyID := int64(41)
	a := &Account{ID: 27, Name: "synthetic", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Status: StatusActive, Schedulable: true, Concurrency: 7, ProxyID: &proxyID,
		GroupIDs: []int64{3, 9}, Groups: []*Group{{ID: 3}},
		Extra:       map[string]any{"codex_fingerprint_seed": "synthetic-stable"},
		Credentials: map[string]any{"access_token": "synthetic-old"}}
	expected := *a
	expected.Credentials = map[string]any{"access_token": "synthetic-new"}
	repo := &providerWriteFallbackRepo{current: a}
	require.NoError(t, (providerWriteAdapter{repo}).UpdateCredentials(context.Background(), a.ID, expected.Credentials))
	require.Equal(t, 1, repo.writes)
	require.Same(t, a, repo.written)
	require.Equal(t, expected, *repo.written)
}

func TestProviderWriteCredentialsReadFailureDoesNotWrite(t *testing.T) {
	for _, readErr := range []error{errors.New("synthetic read failure"), nil} {
		repo := &providerWriteFallbackRepo{readErr: readErr}
		err := (providerWriteAdapter{repo}).UpdateCredentials(context.Background(), 27, map[string]any{"access_token": "synthetic"})
		if readErr == nil {
			readErr = ErrAccountNotFound
		}
		require.ErrorIs(t, err, readErr)
		require.Zero(t, repo.writes)
	}
}

func TestProviderWriteReturnsMergedState(t *testing.T) {
	now := time.Now()
	repo := &providerWriteFallbackRepo{update: func(a *Account) error {
		a.Extra = map[string]any{"managed_state": "current"}
		a.UpdatedAt = now
		return nil
	}}
	record := (&Account{ID: 27, Extra: map[string]any{"managed_state": "stale"}}).ProviderRecord()
	require.NoError(t, (providerWriteAdapter{repo}).Update(context.Background(), record))
	require.Equal(t, "current", record.Extra["managed_state"])
	require.Equal(t, now, record.UpdatedAt)
}
