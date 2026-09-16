//go:build unit

package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func openCodeForkContext(keyID int64) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set("api_key", &APIKey{ID: keyID})
	return c
}

func TestOpenCodeSessionForkTenantAccountAndCredentialIsolation(t *testing.T) {
	account := &Account{ID: 11, Platform: PlatformOpenCodeGo, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "synthetic-upstream"}}
	c := openCodeForkContext(1)
	c.Request.Header.Set(openCodeSessionHeader, "client-conversation")
	first := openCodeScopedSession(c, account)
	require.NotEmpty(t, first)
	require.NotEqual(t, "client-conversation", first)
	require.Equal(t, first, openCodeScopedSession(c, account))
	otherKey := openCodeForkContext(2)
	otherKey.Request.Header.Set(openCodeSessionHeader, "client-conversation")
	require.NotEqual(t, first, openCodeScopedSession(otherKey, account))
	otherAccount := *account
	otherAccount.ID = 12
	require.NotEqual(t, first, openCodeScopedSession(c, &otherAccount))
	account.Credentials = map[string]any{"api_key": "rotated-synthetic-upstream"}
	require.NotEqual(t, first, openCodeScopedSession(c, account))
}

func TestOpenCodeSessionForkSurvivesConversionAndRetry(t *testing.T) {
	account := &Account{ID: 11, Platform: PlatformOpenCodeGo, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "test"}}
	original := []byte(`{"prompt_cache_key":"stable-conversation"}`)
	c := openCodeForkContext(1)
	rememberOpenCodeInboundSession(c, original)
	rememberOpenCodeInboundSession(c, []byte(`{"model":"glm-5.3"}`))
	want := openCodeScopedSession(openCodeForkContext(1), account, original)
	headers := http.Header{}
	applyOpenCodeSessionHeader(c, account, "https://relay.example/v1/messages", headers, []byte(`{"messages":[]}`))
	require.Equal(t, want, headers.Get(openCodeSessionHeader))
	empty := openCodeForkContext(1)
	generated := openCodeScopedSession(empty, account)
	require.Equal(t, generated, openCodeScopedSession(empty, account))
	require.NotEqual(t, generated, openCodeScopedSession(openCodeForkContext(1), account))
	require.Empty(t, openCodeSessionFromBody([]byte(`{"metadata":{"user_id":"only-user-identity"}}`)))
}

func TestOpenCodeSessionForkDoesNotChangeExistingPlatformForwarding(t *testing.T) {
	c := openCodeForkContext(1)
	c.Request.Header.Set(openCodeSessionHeader, "original")
	account := &Account{ID: 11, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	headers := http.Header{}
	applyOpenCodeSessionHeader(c, account, "https://relay.example/v1/responses", headers, []byte(`{"prompt_cache_key":"body"}`))
	require.Empty(t, headers)
	applyOpenCodeSessionHeader(c, account, "https://opencode.ai/zen/go/v1/responses", headers)
	require.Equal(t, "original", headers.Get(openCodeSessionHeader))
}
