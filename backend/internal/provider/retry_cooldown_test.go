//go:build unit

package provider_test

import (
	"context"
	"testing"
	"time"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestCheckErrorPolicy — 6 table-driven cases for the pure logic function
// ---------------------------------------------------------------------------

// TestGatewayFailoverSideEffects_BedrockUsesMappedModel 验证 Bedrock 显式临时规则
// 使用实际上游模型，并禁止池模式同提供商重试。

// ---------------------------------------------------------------------------
// TestApplyErrorPolicy — 4 table-driven cases for the wrapper method
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// errorPolicyRepoStub — minimal ProviderRepository stub for error policy tests
// ---------------------------------------------------------------------------

// retryExhaustedCooldownRepoStub 记录同提供商重试耗尽后的本地冷却写入。
type retryExhaustedCooldownRepoStub struct {
	providercore.RetryCooldownStore

	provider  *providercore.Record
	tempCalls int
}

func (r *retryExhaustedCooldownRepoStub) GetByID(context.Context, int64) (*providercore.Record, error) {
	return r.provider, nil
}

func (r *retryExhaustedCooldownRepoStub) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	r.tempCalls++
	return nil
}

// TestTempUnscheduleRetryableError_PoolModeSkipsLegacyCooldown 验证池模式的
// 同提供商重试耗尽后只切号，不复用旧版 400/502 一分钟冷却。
func TestTempUnscheduleRetryableError_PoolModeSkipsLegacyCooldown(t *testing.T) {
	poolProvider := &providercore.Record{
		LoadLocation: time.LoadLocation, ID: 81,
		Type:     capability.ProviderTypeAPIKey,
		Platform: capability.PlatformAnthropic,
		Credentials: map[string]any{
			"pool_mode": true,
		},
	}
	repo := &retryExhaustedCooldownRepoStub{provider: poolProvider}
	svc := providercore.NewRetryCooldown(repo, providercore.RetryCooldownOptions{})

	svc.Apply(context.Background(), providercore.RetryCooldownInput{ProviderID: poolProvider.ID, Status: 502, Retryable: true})

	require.Zero(t, repo.tempCalls)

	// 非池提供商继续保留旧版特殊错误的冷却行为。
	repo.provider = &providercore.Record{LoadLocation: time.LoadLocation, ID: 82, Type: capability.ProviderTypeOAuth, Platform: capability.PlatformAntigravity}
	svc.Apply(context.Background(), providercore.RetryCooldownInput{ProviderID: repo.provider.ID, Status: 502, Retryable: true})
	require.Equal(t, 1, repo.tempCalls)
}
