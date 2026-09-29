package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	egressadapter "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	upstreamcore "github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestProviderTestService_OpenAIImageOAuthHandlesOutputItemDoneFallback(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &openAIProbeOutput{recorder: rec}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/1/test", nil)

	upstream := &openAIProbeTransport{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
			},
			Body: io.NopCloser(strings.NewReader(
				"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ig_123\",\"type\":\"image_generation_call\",\"result\":\"aGVsbG8=\",\"revised_prompt\":\"draw a cat\",\"output_format\":\"png\"}}\n\n" +
					"data: {\"type\":\"response.completed\",\"response\":{\"created_at\":1710000006,\"tool_usage\":{\"image_gen\":{\"images\":1}},\"output\":[]}}\n\n" +
					"data: [DONE]\n\n",
			)),
		},
	}
	svc := &provideradapter.OpenAIProviderTest{Transport: upstream}
	provider := &providercore.Record{
		ID:       53,
		Name:     "openai-oauth",
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeOAuth,
		Credentials: map[string]any{
			"access_token": "token-123",
		},
	}

	err := executeOpenAIImageOAuthProbe(t, svc, c, context.Background(), provider, "gpt-image-2", "draw a cat")
	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, upstreamcore.HTTPUpstreamProfileOpenAI, upstreamcore.HTTPUpstreamProfileFromContext(upstream.lastReq.Context()))
	require.Contains(t, rec.Body.String(), "Calling Codex /responses image tool")
	require.Contains(t, rec.Body.String(), "data:image/png;base64,aGVsbG8=")
	require.Contains(t, rec.Body.String(), "\"success\":true")
}

func TestProviderTestService_OpenAIImageOAuthForwardsTLSProfile(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &openAIProbeOutput{recorder: rec}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/1/test", nil)

	upstream := &openAIProbeTransport{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(
				"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ig_456\",\"type\":\"image_generation_call\",\"result\":\"aGVsbG8=\",\"output_format\":\"png\"}}\n\n" +
					"data: {\"type\":\"response.completed\",\"response\":{\"output\":[]}}\n\n" +
					"data: [DONE]\n\n",
			)),
		},
	}
	svc := &provideradapter.OpenAIProviderTest{
		Transport:  upstream,
		ResolveTLS: (&provideradapter.OpenAIProbePolicy{ManualProfiles: &egressadapter.TLSProfiles{}}).ResolveTestTLS,
	}
	provider := &providercore.Record{
		ID:       55,
		Name:     "openai-oauth-tls",
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeOAuth,
		Credentials: map[string]any{
			"access_token": "token-123",
		},
		Extra: map[string]any{
			"enable_tls_fingerprint": true,
		},
	}

	err := executeOpenAIImageOAuthProbe(t, svc, c, context.Background(), provider, "gpt-image-2", "draw a cat")
	require.NoError(t, err)
	require.NotNil(t, upstream.lastTLSProfile)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, upstreamcore.HTTPUpstreamProfileOpenAI, upstreamcore.HTTPUpstreamProfileFromContext(upstream.lastReq.Context()))
}

func TestProviderTestService_OpenAIImageAPIKeyUsesConfiguredV1BaseURL(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &openAIProbeOutput{recorder: rec}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/1/test", nil)

	upstream := &openAIProbeTransport{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(strings.NewReader(`{"data":[{"b64_json":"aGVsbG8=","revised_prompt":"draw a cat"}]}`)),
		},
	}
	svc := &provideradapter.OpenAIProviderTest{
		Transport:   upstream,
		ValidateURL: (egress.OperatorURLPolicy{}).Validate,
	}
	provider := &providercore.Record{
		ID:       54,
		Name:     "openai-apikey",
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "test-api-key",
			"base_url": "https://image-upstream.example/v1",
		},
	}

	err := executeOpenAIImageAPIKeyProbe(t, svc, c, context.Background(), provider, "gpt-image-2", "draw a cat")
	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, upstreamcore.HTTPUpstreamProfileOpenAI, upstreamcore.HTTPUpstreamProfileFromContext(upstream.lastReq.Context()))
	require.Equal(t, "https://image-upstream.example/v1/images/generations", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer test-api-key", upstream.lastReq.Header.Get("Authorization"))
	require.Contains(t, rec.Body.String(), "data:image/png;base64,aGVsbG8=")
	require.Contains(t, rec.Body.String(), "\"success\":true")
}

func TestProviderTestService_OpenAIExplicitImageTypeDoesNotInspectModelName(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &openAIProbeOutput{recorder: rec}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/1/test", nil)

	upstream := &openAIProbeTransport{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"b64_json":"aGVsbG8="}]}`)),
		},
	}
	svc := &provideradapter.OpenAIProviderTest{
		Transport:   upstream,
		ValidateURL: (egress.OperatorURLPolicy{}).Validate,
	}
	provider := &providercore.Record{
		ID:       56,
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "test-api-key",
			"base_url": "https://image-upstream.example/v1",
		},
	}

	err := executeOpenAIProbe(t, svc, c, provider, "custom-model-alias", "draw a lighthouse", providercore.ProviderTestModeDefault, providercore.ProviderTestTypeImage)
	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://image-upstream.example/v1/images/generations", upstream.lastReq.URL.String())
	body, err := io.ReadAll(upstream.lastReq.Body)
	require.NoError(t, err)
	require.Equal(t, "custom-model-alias", gjson.GetBytes(body, "model").String())
	require.Equal(t, "draw a lighthouse", gjson.GetBytes(body, "prompt").String())
}

func TestProviderTestService_OpenAIExplicitTextTypeDoesNotInspectModelName(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &openAIProbeOutput{recorder: rec}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/1/test", nil)

	upstream := &openAIProbeTransport{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\"}\n\n")),
		},
	}
	svc := &provideradapter.OpenAIProviderTest{
		Transport:   upstream,
		ValidateURL: (egress.OperatorURLPolicy{}).Validate,
	}
	provider := &providercore.Record{
		ID:       57,
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "test-api-key",
			"base_url": "https://text-upstream.example/v1",
		},
	}

	err := executeOpenAIProbe(t, svc, c, provider, "gpt-image-2", "reply briefly", providercore.ProviderTestModeDefault, providercore.ProviderTestTypeText)
	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://text-upstream.example/v1/responses", upstream.lastReq.URL.String())
	body, err := io.ReadAll(upstream.lastReq.Body)
	require.NoError(t, err)
	require.Equal(t, "gpt-image-2", gjson.GetBytes(body, "model").String())
	require.Equal(t, "reply briefly", gjson.GetBytes(body, "input.0.content.0.text").String())
}
