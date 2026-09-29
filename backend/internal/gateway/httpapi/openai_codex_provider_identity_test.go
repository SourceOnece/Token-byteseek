package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type codexProviderIdentityRepoStub struct {
	gatewayprovider.ExecutionProviderStore

	provider *gatewayprovider.ExecutionProvider
}

func (s *codexProviderIdentityRepoStub) GetByID(_ context.Context, _ int64) (*gatewayprovider.ExecutionProvider, error) {
	return s.provider, nil
}

func TestCodexProviderIdentitySourceResolvesShadowAndOverwritesFailoverContext(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	parentID := int64(11)
	parent := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: parentID, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Credentials: map[string]any{
		"chatgpt_account_id": "team-provider",
		"chatgpt_user_id":    "user-1",
	}}}
	shadow := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 111, ParentProviderID: &parentID, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}}
	service := newResponsesFixture(responsesFixtureInputs{providers: &codexProviderIdentityRepoStub{provider: parent}})

	resolved, err := PrepareCodexIdentity(context.Background(), c, service.Requests.Providers, shadow)
	require.NoError(t, err)
	require.Same(t, parent.View(), resolved)
	require.Same(t, parent.View(), CodexIdentityRecord(c, shadow.View()))

	req, err := service.Requests.Build(
		context.Background(), c, shadow,
		[]byte(`{"model":"gpt-5.6-codex","stream":true,"prompt_cache_key":"client-session"}`),
		"token", true, "client-session", true,
	)
	require.NoError(t, err)
	require.Equal(t, openai.IsolateOpenAIUpstreamSessionID(0, provideradapter.CodexIdentityNamespace(parent.View()), "client-session"), req.Header.Get("session_id"))

	next := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 19, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Credentials: map[string]any{
		"chatgpt_account_id": "other-provider",
		"chatgpt_user_id":    "user-2",
	}}}
	resolved, err = PrepareCodexIdentity(context.Background(), c, service.Requests.Providers, next)
	require.NoError(t, err)
	require.Same(t, next.View(), resolved)
	require.Same(t, next.View(), CodexIdentityRecord(c, shadow.View()))
}

func TestBuildUpstreamRequestNamespacesCodexIdentityByOAuthProvider(t *testing.T) {
	svc := newResponsesFixture(responsesFixtureInputs{})
	body := []byte(`{"model":"gpt-5.6-codex","stream":true,"prompt_cache_key":"client-session"}`)

	build := func(providerID int64, chatgptAccountID string) http.Header {
		t.Helper()
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
		c.Set("api_key", &apikey.APIKey{ID: 77})
		c.Request.Header.Set("User-Agent", "codex_cli_rs/0.144.0")
		c.Request.Header.Set("x-codex-installation-id", "client-installation")
		c.Request.Header.Set("x-codex-window-id", "client-window")
		c.Request.Header.Set("session-id", "client-session")
		c.Request.Header.Set("thread-id", "client-thread")
		c.Request.Header.Set("x-client-request-id", "client-request")
		c.Request.Header.Set("x-codex-turn-metadata", `{"installation_id":"client-installation","session_id":"client-session","thread_id":"client-thread","turn_id":"client-turn","window_id":"client-window"}`)

		provider := &gatewayprovider.ExecutionProvider{
			Record: providercore.Record{
				LoadLocation: time.LoadLocation, ID: providerID,
				Platform: capability.PlatformOpenAI,
				Type:     capability.ProviderTypeOAuth,
				Credentials: map[string]any{
					"chatgpt_account_id": chatgptAccountID,
				},
			},
		}
		req, err := svc.Requests.Build(
			context.Background(), c, provider, body, "oauth-token", true, "client-session", true,
		)
		require.NoError(t, err)
		return req.Header
	}

	first := build(11, "chatgpt-provider-11")
	firstAgain := build(11, "chatgpt-provider-11")
	second := build(19, "chatgpt-provider-19")

	identityHeaders := []string{
		"x-codex-installation-id",
		"x-codex-window-id",
		"session-id",
		"session_id",
		"conversation_id",
		"thread-id",
		"x-client-request-id",
		"x-codex-turn-metadata",
	}
	checked := 0
	for _, header := range identityHeaders {
		if first.Get(header) == "" && second.Get(header) == "" {
			continue
		}
		checked++
		require.NotEmpty(t, first.Get(header), header)
		require.Equal(t, first.Get(header), firstAgain.Get(header), "same provider must retain stable identity: %s", header)
		require.NotEqual(t, first.Get(header), second.Get(header), "provider failover must rotate upstream identity: %s", header)
	}
	require.GreaterOrEqual(t, checked, 5, "test must exercise the real outbound identity surface")
}
