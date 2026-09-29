package provider_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func TestOllamaCloudUsageManagedExtraCannotBeImported(t *testing.T) {
	remoteExtra := map[string]any{
		providercore.OllamaCloudUsageSessionExtraKey:     "remote-ciphertext",
		providercore.OllamaCloudUsageAutoRefreshExtraKey: true,
		providercore.OllamaCloudUsageSnapshotExtraKey:    map[string]any{"status": "forged"},
	}
	created, err := providercore.BuildProviderForCreate(&providercore.CreateProviderInput{
		Name: "ollama", Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://ollama.com", "api_key": "key"},
		Concurrency: 1,
	}, providercore.CRSMergeMap(nil, remoteExtra), providercore.CreationOptions{Now: time.Now, LoadLocation: time.LoadLocation, NewSeed: uuid.NewString})
	require.NoError(t, err)
	require.NotContains(t, created.Extra, providercore.OllamaCloudUsageSessionExtraKey)
	require.NotContains(t, created.Extra, providercore.OllamaCloudUsageAutoRefreshExtraKey)
	require.NotContains(t, created.Extra, providercore.OllamaCloudUsageSnapshotExtraKey)

	existing := ollamaUsageProvider(6)
	existing.Extra = map[string]any{
		providercore.OllamaCloudUsageSessionExtraKey:     "local-ciphertext",
		providercore.OllamaCloudUsageAutoRefreshExtraKey: false,
		providercore.OllamaCloudUsageSnapshotExtraKey:    map[string]any{"status": providercore.OllamaCloudUsageStatusOK},
	}
	targetExtra := providercore.CRSMergeMap(existing.Extra, remoteExtra)
	providercore.ReconcileCRSOllamaCloudUsageExtra(existing, existing.Platform, existing.Type, providercore.CRSMergeMap(existing.Credentials, nil), targetExtra)
	require.Equal(t, "local-ciphertext", targetExtra[providercore.OllamaCloudUsageSessionExtraKey])
	require.Equal(t, false, targetExtra[providercore.OllamaCloudUsageAutoRefreshExtraKey])
	require.Equal(t, map[string]any{"status": providercore.OllamaCloudUsageStatusOK}, targetExtra[providercore.OllamaCloudUsageSnapshotExtraKey])

	changedCredentials := providercore.CRSMergeMap(existing.Credentials, map[string]any{"api_key": "rotated"})
	targetExtra = providercore.CRSMergeMap(existing.Extra, remoteExtra)
	providercore.ReconcileCRSOllamaCloudUsageExtra(existing, existing.Platform, existing.Type, changedCredentials, targetExtra)
	require.NotContains(t, targetExtra, providercore.OllamaCloudUsageSessionExtraKey)
	require.NotContains(t, targetExtra, providercore.OllamaCloudUsageAutoRefreshExtraKey)
	require.NotContains(t, targetExtra, providercore.OllamaCloudUsageSnapshotExtraKey)
}

func ollamaUsageProvider(id int64) *providercore.Record {
	return &providercore.Record{
		ID: id, Name: fmt.Sprintf("ollama-%d", id), Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://ollama.com", "api_key": fmt.Sprintf("key-%d", id)},
		Extra:       map[string]any{}, Status: billing.StatusActive, Schedulable: true, Concurrency: 1,
	}
}
