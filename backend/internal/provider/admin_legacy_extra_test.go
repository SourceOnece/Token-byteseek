package provider_test

import (
	"context"
	"net/http"
	"testing"

	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	billingcore "github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	xai "github.com/TokenFlux/TokenRouter/internal/upstream/grok"
	"github.com/stretchr/testify/require"
)

func TestCreateProviderDiscardsDeprecatedBillingProbeExtra(t *testing.T) {
	repo := &providerServiceTestRepo{}
	created, err := newProviderEditorForTest(repo).CreateProvider(context.Background(), &provider.CreateProviderInput{
		Name:        "upstream",
		Platform:    capability.PlatformOpenAI,
		Type:        capability.ProviderTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra: map[string]any{
			"upstream_billing_probe_enabled": true,
			"upstream_billing_probe":         map[string]any{"status": "ok"},
			"custom":                         "value",
		},
	})

	require.NoError(t, err)
	require.NotContains(t, created.Extra, "upstream_billing_probe_enabled")
	require.NotContains(t, created.Extra, "upstream_billing_probe")
	require.Equal(t, "value", created.Extra["custom"])
}

func TestUpdateProviderDiscardsDeprecatedBillingProbeExtra(t *testing.T) {
	providerID := int64(110)
	repo := &providerServiceTestRepo{providers: map[int64]*provider.Record{
		providerID: {
			ID:       providerID,
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Status:   billingcore.StatusActive,
			Extra: map[string]any{
				"upstream_billing_probe_enabled": true,
				"upstream_billing_probe":         map[string]any{"status": "ok"},
			},
		},
	}}

	updated, err := newProviderEditorForTest(repo).UpdateProvider(context.Background(), providerID, &provider.UpdateProviderInput{
		Extra: map[string]any{
			"upstream_billing_probe_enabled": false,
			"upstream_billing_probe":         map[string]any{"status": "forged"},
			"custom":                         "value",
		},
	})

	require.NoError(t, err)
	require.NotContains(t, updated.Extra, "upstream_billing_probe_enabled")
	require.NotContains(t, updated.Extra, "upstream_billing_probe")
	require.Equal(t, "value", updated.Extra["custom"])
}

func TestBulkUpdateProvidersDiscardsDeprecatedBillingProbeExtra(t *testing.T) {
	repo := &providerServiceTestRepo{}
	result, err := newProviderEditorForTest(repo).BulkUpdateProviders(context.Background(), &provider.BulkUpdateProvidersInput{
		ProviderIDs: []int64{1},
		Extra: map[string]any{
			"upstream_billing_probe_enabled": true,
			"upstream_billing_probe":         map[string]any{"status": "ok"},
			"custom":                         "value",
		},
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Success)
	require.Len(t, repo.bulkUpdates, 1)
	require.NotContains(t, repo.bulkUpdates[0].Extra, "upstream_billing_probe_enabled")
	require.NotContains(t, repo.bulkUpdates[0].Extra, "upstream_billing_probe")
	require.Equal(t, "value", repo.bulkUpdates[0].Extra["custom"])
}

func TestUpdateProviderPreservesGrokBillingSnapshotForUnrelatedEdit(t *testing.T) {
	providerID := int64(112)
	billing := &xai.BillingSummary{
		StatusCode:       http.StatusForbidden,
		WeeklyStatusCode: http.StatusForbidden,
	}
	repo := &providerServiceTestRepo{providers: map[int64]*provider.Record{
		providerID: {
			ID:       providerID,
			Platform: capability.PlatformGrok,
			Type:     capability.ProviderTypeOAuth,
			Status:   billingcore.StatusActive,
			Extra:    map[string]any{provider.GrokUsageBillingExtraKey: billing},
		},
	}}

	updated, err := newProviderEditorForTest(repo).UpdateProvider(context.Background(), providerID, &provider.UpdateProviderInput{
		Extra: map[string]any{"custom": "value"},
	})

	require.NoError(t, err)
	require.Equal(t, billing, updated.Extra[provider.GrokUsageBillingExtraKey])
	require.Equal(t, "value", updated.Extra["custom"])
	eligible, reason := provider.GrokMediaGenerationEligibility(updated, provideradapter.GrokTierRules())
	require.False(t, eligible)
	require.Equal(t, "billing_forbidden", reason)
}
