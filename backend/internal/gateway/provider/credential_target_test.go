package provider

import (
	"context"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

// stubCredRepo 是最小化 ProviderRepository stub，仅实现 GetByID，供 credential_shadow_test 使用。
// 嵌入接口满足完整方法集；未实现的方法若被调用会 panic，从而快速暴露误调用。
type stubCredRepo struct {
	ExecutionProviderStore

	parent *ExecutionProvider
}

func (s *stubCredRepo) GetByID(_ context.Context, _ int64) (*ExecutionProvider, error) {
	return s.parent, nil
}

func newStubCredRepo(parent *ExecutionProvider) ExecutionProviderStore {
	return &stubCredRepo{parent: parent}
}

func TestResolveCredentialProvider(t *testing.T) {
	ctx := context.Background()
	pid := int64(100)

	// 普通提供商（非影子）→ 返回自身
	parent := &ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 100, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive}}
	repo := newStubCredRepo(parent)
	got, err := CredentialProvider(ctx, repo, parent)
	require.NoError(t, err)
	require.Equal(t, int64(100), got.Record.ID)

	// 影子提供商 + 合法 OpenAI OAuth 母提供商 → 返回母提供商
	shadow := &ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 200, ParentProviderID: &pid, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}}
	got, err = CredentialProvider(ctx, repo, shadow)
	require.NoError(t, err)
	require.Equal(t, int64(100), got.Record.ID)

	// 影子提供商 + 母提供商非 OpenAI OAuth（API Key 类型）→ 返回 error
	badRepo := newStubCredRepo(&ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 100, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey}})
	_, err = CredentialProvider(ctx, badRepo, shadow)
	require.Error(t, err)
}
