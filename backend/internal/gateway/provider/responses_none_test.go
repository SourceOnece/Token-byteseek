package provider

import (
	"testing"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestFilterOpenAIResponsesNoneReasoningEffortForProvider(t *testing.T) {
	tests := []struct {
		name          string
		provider      *providercore.Record
		body          string
		wantNested    bool
		wantFlat      bool
		wantSummary   bool
		wantReasoning bool
	}{
		{
			name:          "Kimi removes none placeholder",
			provider:      &providercore.Record{Platform: capability.PlatformKimi, Type: capability.ProviderTypeAPIKey},
			body:          `{"reasoning":{"effort":"none"},"reasoning_effort":"NONE"}`,
			wantReasoning: false,
		},
		{
			name:          "custom compatible endpoint removes none placeholder",
			provider:      &providercore.Record{Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey, Credentials: map[string]any{"base_url": "https://compat.example/v1"}},
			body:          `{"reasoning":{"effort":"none"},"reasoning_effort":"NONE"}`,
			wantReasoning: false,
		},
		{
			name:          "preserves other reasoning members",
			provider:      &providercore.Record{Platform: capability.PlatformGrok, Type: capability.ProviderTypeAPIKey},
			body:          `{"reasoning":{"effort":" none ","summary":"auto"}}`,
			wantSummary:   true,
			wantReasoning: true,
		},
		{
			name:          "official OpenAI API preserves none",
			provider:      &providercore.Record{Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey},
			body:          `{"reasoning":{"effort":"none"},"reasoning_effort":"none"}`,
			wantNested:    true,
			wantFlat:      true,
			wantReasoning: true,
		},
		{
			name:          "OpenAI OAuth preserves none",
			provider:      &providercore.Record{Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth},
			body:          `{"reasoning":{"effort":"none"}}`,
			wantNested:    true,
			wantReasoning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FilterOpenAIResponsesNoneReasoningEffortForProvider(tt.provider, []byte(tt.body))
			require.NoError(t, err)
			require.Equal(t, tt.wantNested, gjson.GetBytes(got, "reasoning.effort").Exists())
			require.Equal(t, tt.wantFlat, gjson.GetBytes(got, "reasoning_effort").Exists())
			require.Equal(t, tt.wantSummary, gjson.GetBytes(got, "reasoning.summary").Exists())
			require.Equal(t, tt.wantReasoning, gjson.GetBytes(got, "reasoning").Exists())
		})
	}
}

func TestFilterOpenAIResponsesNoneReasoningEffortForProvider_APIKeyAutomaticPassthroughPreservesRequest(t *testing.T) {
	body := []byte(`{"model":"qwen3.8-27b","input":"hi","max_output_tokens":20,"reasoning":{"effort":"none"},"presence_penalty":1.5}`)
	provider := &providercore.Record{
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://compat.example/v1",
		},
		Extra: map[string]any{"openai_passthrough": true},
	}

	got, err := FilterOpenAIResponsesNoneReasoningEffortForProvider(provider, body)

	require.NoError(t, err)
	require.JSONEq(t, string(body), string(got))
}
