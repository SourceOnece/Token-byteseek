package provider_test

import (
	"context"
	"net/http"
	"testing"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	"github.com/TokenFlux/TokenRouter/internal/server/httpx"
	"github.com/stretchr/testify/require"
)

func TestProtocolSaveEntrypointsAndBulkRejectBeforeWrite(t *testing.T) {
	repo := &providerServiceTestRepo{providers: map[int64]*providercore.Record{}}
	svc := newProviderEditorForTest(repo)
	created, err := svc.CreateProvider(context.Background(), &providercore.CreateProviderInput{Name: "native", Platform: capability.PlatformKimi, Type: capability.ProviderTypeAPIKey, Credentials: map[string]any{"api_key": "test", providercore.UpstreamProtocolsKey: []string{"anthropic_messages", "openai_responses", "openai_chat_completions"}, "api_base_urls": map[string]any{"responses": "https://relay.example/v1"}}})
	require.NoError(t, err)
	require.NotContains(t, created.Credentials, "api_protocol")
	require.Len(t, created.UpstreamProtocols(), 3)
	updated, err := svc.UpdateProvider(context.Background(), created.ID, &providercore.UpdateProviderInput{Name: "renamed"})
	require.NoError(t, err)
	require.Len(t, updated.UpstreamProtocols(), 3)
	require.Equal(t, map[string]any{"responses": "https://relay.example/v1"}, updated.Credentials["api_base_urls"])
	rotated, err := svc.UpdateProvider(context.Background(), created.ID, &providercore.UpdateProviderInput{Credentials: map[string]any{"api_key": "rotated"}})
	require.NoError(t, err)
	require.Len(t, rotated.UpstreamProtocols(), 3)
	require.Equal(t, map[string]any{"responses": "https://relay.example/v1"}, rotated.Credentials["api_base_urls"])
	repo.providers[99] = &providercore.Record{ID: 99, Platform: capability.PlatformZhipu, Type: capability.ProviderTypeAPIKey, Credentials: map[string]any{providercore.UpstreamProtocolsKey: []string{"openai_chat_completions"}}}
	_, err = svc.BulkUpdateProviders(context.Background(), &providercore.BulkUpdateProvidersInput{ProviderIDs: []int64{created.ID, 99}, Credentials: map[string]any{providercore.UpstreamProtocolsKey: []string{"openai_responses"}}})
	require.Equal(t, http.StatusBadRequest, httpx.ErrorCode(err))
	require.Empty(t, repo.bulkUpdates)
}
