package provider_test

import (
	"context"
	"testing"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/stretchr/testify/require"
)

func TestAdminUpdatePreservesOllamaManagedExtra(t *testing.T) {
	provider := ollamaUsageProvider(61)
	provider.Extra = map[string]any{
		providercore.OllamaCloudUsageSessionExtraKey:     "local-ciphertext",
		providercore.OllamaCloudUsageAutoRefreshExtraKey: true,
		providercore.OllamaCloudUsageSnapshotExtraKey:    map[string]any{"status": providercore.OllamaCloudUsageStatusOK},
	}
	repo := &ollamaManagedExtraUpdateRepo{provider: provider}
	svc := newProviderEditorForTest(repo)
	requestedExtra := map[string]any{
		"note": "preserved",
		providercore.OllamaCloudUsageSessionExtraKey:     "forged-ciphertext",
		providercore.OllamaCloudUsageAutoRefreshExtraKey: nil,
		providercore.OllamaCloudUsageSnapshotExtraKey:    nil,
	}

	_, err := svc.UpdateProvider(context.Background(), provider.ID, &providercore.UpdateProviderInput{Extra: requestedExtra})
	require.NoError(t, err)
	require.Equal(t, "preserved", repo.updated.Extra["note"])
	require.Equal(t, "local-ciphertext", repo.updated.Extra[providercore.OllamaCloudUsageSessionExtraKey])
	require.Equal(t, true, repo.updated.Extra[providercore.OllamaCloudUsageAutoRefreshExtraKey])
	require.Equal(t, provider.Extra[providercore.OllamaCloudUsageSnapshotExtraKey], repo.updated.Extra[providercore.OllamaCloudUsageSnapshotExtraKey])

	require.Contains(t, requestedExtra, providercore.OllamaCloudUsageSessionExtraKey)
}

type ollamaManagedExtraUpdateRepo struct {
	providercore.AdminStore
	provider *providercore.Record
	updated  *providercore.Record
}

func (r *ollamaManagedExtraUpdateRepo) GetByID(_ context.Context, _ int64) (*providercore.Record, error) {
	return providercore.CloneRecord(r.provider), nil
}

func (r *ollamaManagedExtraUpdateRepo) Update(_ context.Context, provider *providercore.Record) error {
	r.updated = providercore.CloneRecord(provider)
	return nil
}
