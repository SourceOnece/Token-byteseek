package httpapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	keyhttp "github.com/TokenFlux/TokenRouter/internal/apikey/httpapi"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 同一对话跨协议稳定，切 Key、提供商或凭据后必须隔离；中继只接受显式平台的派生值。
func TestOpenCodeScopedSessionRetainsForkIsolation(t *testing.T) {
	c := newOpenCodeSessionTestContext(t, "")
	c.Set(string(keyhttp.ContextKeyAPIKey), &apikey.APIKey{ID: 7, UserID: 2})
	value := openCodeSessionTestProvider("https://relay.invalid/zen/go/v1")
	value.Record.Platform = capability.PlatformOpenCodeGo
	value.Record.Credentials["api_key"] = "fixture-a"
	body := []byte(`{"model":"gpt-5.6-luna","input":"hello","prompt_cache_key":"conversation-a"}`)
	rememberOpenCodeInboundSession(c, body)
	first := openCodeScopedSession(c, value)
	require.NotEmpty(t, first)
	require.NotEqual(t, "conversation-a", first)
	svc := openCodeSessionTestService()
	req, _, err := svc.Requests.BuildAnthropic(context.Background(), c, value, []byte(`{"model":"qwen","messages":[]}`), "fixture-a", "https://relay.invalid/zen/go/v1/messages")
	require.NoError(t, err)
	require.Equal(t, first, req.Header.Get(openCodeSessionHeader))
	require.Equal(t, "opencode/1.0.0", req.Header.Get("User-Agent"))
	c.Set(string(keyhttp.ContextKeyAPIKey), &apikey.APIKey{ID: 8, UserID: 2})
	require.NotEqual(t, first, openCodeScopedSession(c, value))
	c.Set(string(keyhttp.ContextKeyAPIKey), &apikey.APIKey{ID: 7, UserID: 2})
	value.Record.ID++
	require.NotEqual(t, first, openCodeScopedSession(c, value))
	value.Record.ID--
	value.Record.Credentials["api_key"] = "fixture-b"
	require.NotEqual(t, first, openCodeScopedSession(c, value))
	value.Record.Platform = capability.PlatformOpenAI
	h := http.Header{}
	ApplyOpenCodeSessionHeader(c, value, "https://relay.invalid/v1/responses", h)
	require.Empty(t, h.Get(openCodeSessionHeader))
}

// 普通 user_id 不可合并所有会话；缺少会话时，同次重试稳定、不同请求随机。
func TestOpenCodeMissingSessionIsRequestScoped(t *testing.T) {
	value := openCodeSessionTestProvider("https://opencode.ai/zen/go/v1")
	value.Record.Platform = capability.PlatformOpenCodeGo
	one := newOpenCodeSessionTestContext(t, "")
	two := newOpenCodeSessionTestContext(t, "")
	for _, c := range []*gin.Context{one, two} {
		rememberOpenCodeInboundSession(c, []byte(`{"metadata":{"user_id":"same-user"}}`))
	}
	first := openCodeScopedSession(one, value)
	require.Equal(t, first, openCodeScopedSession(one, value))
	require.NotEqual(t, first, openCodeScopedSession(two, value))
}
