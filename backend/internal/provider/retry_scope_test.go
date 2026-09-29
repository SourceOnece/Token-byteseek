package provider_test

import (
	"context"
	"testing"
	"time"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	"github.com/stretchr/testify/require"
)

type capacityShedProviderRepoStub struct {
	providercore.RetryCooldownStore
	// 嵌入接口，未实现的方法会 panic（不应被调用）

	tempUnschedCalls int
}

func (r *capacityShedProviderRepoStub) SetTempUnschedulable(_ context.Context, _ int64, _ time.Time, _ string) error {
	r.tempUnschedCalls++
	return nil
}

func (r *capacityShedProviderRepoStub) GetByID(_ context.Context, id int64) (*providercore.Record, error) {
	return &providercore.Record{LoadLocation: time.LoadLocation, ID: id, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}, nil
}

func TestTempUnscheduleRetryableErrorSkipsRequestScopedTransient(t *testing.T) {
	t.Run("请求级瞬时故障不写提供商状态", func(t *testing.T) {
		repo := &capacityShedProviderRepoStub{}
		svc := providercore.NewRetryCooldown(repo, providercore.RetryCooldownOptions{})

		svc.Apply(context.Background(), providercore.RetryCooldownInput{ProviderID: 1, Status: 502, Retryable: true, RequestScopedTransient: true})

		require.Zero(t, repo.tempUnschedCalls)
	})

	// 对照组：同样的 502 在未标记请求级瞬时故障时仍按原有语义临时摘号，
	// 确认上面的断言来自新增守卫而非其他前置条件。
	t.Run("未标记时保持原有临时摘号语义", func(t *testing.T) {
		repo := &capacityShedProviderRepoStub{}
		svc := providercore.NewRetryCooldown(repo, providercore.RetryCooldownOptions{})

		svc.Apply(context.Background(), providercore.RetryCooldownInput{ProviderID: 1, Status: 502, Retryable: true})

		require.Equal(t, 1, repo.tempUnschedCalls)
	})
}
