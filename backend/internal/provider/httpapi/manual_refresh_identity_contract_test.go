package httpapi

import (
	"context"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

// 模拟手动交换返回前管理员换凭据；旧交换结果必须让位，不能重新覆盖。
func TestManualRefreshDoesNotOverwriteNewAdministratorCredentials(t *testing.T) {
	adminSvc := newManagementMutationFixture()
	adminSvc.providers = []provider.Record{{ID: 977, Platform: capability.PlatformQoder, Type: capability.ProviderTypeCosy, Status: billing.StatusActive, Credentials: map[string]any{"refresh_token": "observed", "security_oauth_token": "old", "machine_id": "machine"}}}
	observed := adminSvc.providers[0]
	exchange := &provider.ManualCredentialExchange{Qoder: func(context.Context, *provider.Record) (map[string]any, error) {
		adminSvc.providers[0].Credentials = map[string]any{"refresh_token": "administrator", "security_oauth_token": "administrator-token", "machine_id": "machine"}
		return map[string]any{"refresh_token": "late", "security_oauth_token": "late-token", "machine_id": "machine"}, nil
	}}
	h := newManagedRefreshFixture(adminSvc, exchange)
	updated, _, err := h.Refresh(context.Background(), &observed)
	require.NoError(t, err)
	require.Nil(t, adminSvc.updateProviderInput, "迟到交换不应提交旧凭据")
	require.Equal(t, "administrator", updated.GetCredential("refresh_token"))
}
