package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/creative"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient/tlsfingerprint"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/stretchr/testify/require"
)

type creativeOAuthRequests struct {
	profile *tlsfingerprint.Profile
	matched int
}

func (r *creativeOAuthRequests) ImagesURL(*ExecutionProvider, string) (string, error) {
	return "https://api.example.com/v1/images/generations", nil
}
func (r *creativeOAuthRequests) ValidateBaseURL(url string) (string, error) { return url, nil }
func (r *creativeOAuthRequests) MatchTLSInput(agent func() string, _ *ExecutionProvider) egress.TLSFingerprintRouterMatchResult {
	r.matched++
	return egress.TLSFingerprintRouterMatchResult{Matched: true, UpstreamUserAgent: agent()}
}

func (r *creativeOAuthRequests) TLSProfile(_ *ExecutionProvider, match ...egress.TLSFingerprintRouterMatchResult) *tlsfingerprint.Profile {
	return r.profile
}

type creativeOAuthTransport struct {
	call func(*http.Request, string, int64, int, *tlsfingerprint.Profile) (*http.Response, error)
}

func (r creativeOAuthTransport) Do(req *http.Request, proxy string, id int64, limit int) (*http.Response, error) {
	return r.call(req, proxy, id, limit, nil)
}

func (r creativeOAuthTransport) DoWithTLS(req *http.Request, proxy string, id int64, limit int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return r.call(req, proxy, id, limit, profile)
}

// 通过生产 ForProvider 装配执行任务，覆盖认证分派、账号代理和 TLS 参数。
func TestCreativeOAuthProductionTarget(t *testing.T) {
	requests := &creativeOAuthRequests{profile: &tlsfingerprint.Profile{}}
	proxyID := int64(3)
	value := NewExecutionProvider(&providercore.Record{
		ID: 7, Platform: "openai", Type: "oauth", Concurrency: 4,
		Credentials: map[string]any{"access_token": "fixture-token", "chatgpt_account_id": "account-a"},
		ProxyID:     &proxyID, Proxy: &egress.Proxy{Protocol: "http", Host: "proxy.example.com", Port: 8888},
	})
	var sessions []string
	targets := &CreativeTargets{
		Requests: requests, Credentials: &providercore.OpenAIExecutionCredentials{}, Identity: &ExecutionAgentIdentity{}, ClientPolicy: &provideradapter.OpenAIProbePolicy{},
		Transport: creativeOAuthTransport{call: func(req *http.Request, proxy string, id int64, limit int, profile *tlsfingerprint.Profile) (*http.Response, error) {
			require.Equal(t, "https://chatgpt.com/backend-api/codex/images/generations", req.URL.String())
			require.Equal(t, "chatgpt.com", req.Host)
			require.Equal(t, "Bearer fixture-token", req.Header.Get("Authorization"))
			require.Equal(t, value.View().GetChatGPTAccountID(), req.Header.Get("ChatGPT-Account-Id"))
			require.Equal(t, value.Record.Proxy.URL(), proxy)
			require.Equal(t, int64(7), id)
			require.Equal(t, 4, limit)
			require.Same(t, requests.profile, profile)
			require.Equal(t, openai.CodexCanonicalUserAgent(), req.Header.Get("User-Agent"))
			require.NotEmpty(t, req.Header.Get("session_id"))
			sessions = append(sessions, req.Header.Get("session_id"))
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"b64_json":"aW1hZ2U="}]}`))}, nil
		}},
	}
	for _, run := range []creative.CreativeRun{{UserID: 1, APIKeyID: 10, RunID: "run-a"}, {UserID: 2, APIKeyID: 20, RunID: "run-a"}, {UserID: 1, APIKeyID: 10, RunID: "run-b"}} {
		run.Operation = "generate"
		run.ImageSize = "1K"
		outputs, err := targets.ForProvider(value).ExecutePlatform(context.Background(), "openai", run, creative.CreativeRunPayload{Prompt: "cat"}, "gpt-image-2")
		require.NoError(t, err)
		require.Len(t, outputs, 1)
	}
	require.Equal(t, 3, requests.matched)
	require.NotEqual(t, sessions[0], sessions[1])
	require.NotEqual(t, sessions[0], sessions[2])
	// 相同任务换号时，即使缺少 ChatGPT 账号标识也使用不同会话。
	run := creative.CreativeRun{UserID: 1, APIKeyID: 10, RunID: "run-a"}
	for _, credentials := range []map[string]any{{"access_token": "fixture-token", "chatgpt_account_id": "account-a"}, {"access_token": "fixture-token"}} {
		value.Record.Credentials = credentials
		first, err := targets.creativeOAuthRequest(context.Background(), value, run, []byte(`{}`), "fixture-token", "https://chatgpt.com/backend-api/codex/responses", egress.TLSFingerprintRouterMatchResult{})
		require.NoError(t, err)
		value.Record.ID = 8
		value.Record.Credentials = map[string]any{"access_token": "fixture-token", "chatgpt_account_id": "account-b"}
		second, err := targets.creativeOAuthRequest(context.Background(), value, run, []byte(`{}`), "fixture-token", "https://chatgpt.com/backend-api/codex/responses", egress.TLSFingerprintRouterMatchResult{})
		require.NoError(t, err)
		require.NotEqual(t, first.Header.Get("session_id"), second.Header.Get("session_id"))
		value.Record.ID = 7
	}
}

func TestCreativeOtherPlatformsDoNotSelectCodexTLS(t *testing.T) {
	for _, kind := range []struct{ platform, typ string }{{"openai", "apikey"}, {"grok", "oauth"}, {"gemini", "oauth"}} {
		requests := &creativeOAuthRequests{}
		target := (&CreativeTargets{Requests: requests}).ForProvider(NewExecutionProvider(&providercore.Record{Platform: kind.platform, Type: kind.typ}))
		require.False(t, target.OpenAI.OAuth)
		require.Zero(t, requests.matched)
	}
}

type creativeParentReader struct{ parent *ExecutionProvider }

func (r creativeParentReader) GetByID(context.Context, int64) (*ExecutionProvider, error) {
	return r.parent, nil
}

// 影子账号使用母账号的 ChatGPT 身份，指纹收敛模式仍由已有配置决定。
func TestCreativeOAuthShadowCredentialAndFingerprint(t *testing.T) {
	parentID := int64(11)
	parent := NewExecutionProvider(&providercore.Record{ID: parentID, Platform: "openai", Type: "oauth", Credentials: map[string]any{"chatgpt_account_id": "parent-account", "access_token": "parent-token"}, Extra: map[string]any{"codex_fingerprint_mode": "device", "codex_fingerprint_seed": "11111111-1111-4111-8111-111111111111"}})
	shadow := NewExecutionProvider(&providercore.Record{ID: 111, ParentProviderID: &parentID, Platform: "openai", Type: "oauth"})
	reader := creativeParentReader{parent: parent}
	targets := &CreativeTargets{Providers: reader, Identity: &ExecutionAgentIdentity{}, ClientPolicy: &provideradapter.OpenAIProbePolicy{}}
	run := creative.CreativeRun{UserID: 1, APIKeyID: 10, RunID: "run-a"}
	first, err := targets.creativeOAuthRequest(context.Background(), shadow, run, []byte(`{}`), "parent-token", "https://chatgpt.com/backend-api/codex/images/generations", egress.TLSFingerprintRouterMatchResult{})
	require.NoError(t, err)
	require.Equal(t, "parent-account", first.Header.Get("ChatGPT-Account-Id"))
	require.NotEmpty(t, first.Header.Get("x-codex-installation-id"))
	run.UserID = 2
	run.APIKeyID = 20
	second, err := targets.creativeOAuthRequest(context.Background(), shadow, run, []byte(`{}`), "parent-token", "https://chatgpt.com/backend-api/codex/images/generations", egress.TLSFingerprintRouterMatchResult{})
	require.NoError(t, err)
	require.Equal(t, first.Header.Get("x-codex-installation-id"), second.Header.Get("x-codex-installation-id"))
	require.NotEqual(t, first.Header.Get("session_id"), second.Header.Get("session_id"))
}
