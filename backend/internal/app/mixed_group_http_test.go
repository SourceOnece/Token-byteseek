package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	keyhttp "github.com/TokenFlux/TokenRouter/internal/apikey/httpapi"
	keytestkit "github.com/TokenFlux/TokenRouter/internal/apikey/testkit"
	"github.com/TokenFlux/TokenRouter/internal/config"
	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/googleforward"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/messageforward"
	"github.com/TokenFlux/TokenRouter/internal/gateway/requeststate"
	"github.com/TokenFlux/TokenRouter/internal/gateway/searchtools"
	gatewaytestkit "github.com/TokenFlux/TokenRouter/internal/gateway/testkit"
	"github.com/TokenFlux/TokenRouter/internal/identity"
	"github.com/TokenFlux/TokenRouter/internal/identity/httpapi/authctx"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient/tlsfingerprint"
	"github.com/TokenFlux/TokenRouter/internal/protocol"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// mixedHTTPProviders 只提供持久提供商查询，资格与排序使用真实选择器。
type mixedHTTPProviders struct {
	gatewayadapter.ExecutionProviderStore
	values []gatewayadapter.ExecutionProvider
}

func (*mixedHTTPProviders) completeGroupProjection() {}

func (s *mixedHTTPProviders) GetByID(_ context.Context, id int64) (*gatewayadapter.ExecutionProvider, error) {
	for _, value := range s.values {
		if value.Record.ID == id {
			return gatewayadapter.NewExecutionProvider(&value.Record), nil
		}
	}
	return nil, nil
}

func (s *mixedHTTPProviders) GetByIDs(ctx context.Context, ids []int64) ([]*gatewayadapter.ExecutionProvider, error) {
	var out []*gatewayadapter.ExecutionProvider
	for _, id := range ids {
		value, _ := s.GetByID(ctx, id)
		if value != nil {
			out = append(out, value)
		}
	}
	return out, nil
}

func (s *mixedHTTPProviders) ListSchedulableByGroupIDAndPlatforms(_ context.Context, id int64, platforms []string) ([]gatewayadapter.ExecutionProvider, error) {
	var out []gatewayadapter.ExecutionProvider
	for _, value := range s.values {
		if slices.Contains(value.Record.GroupIDs, id) && slices.Contains(platforms, value.Record.Platform) {
			out = append(out, *gatewayadapter.NewExecutionProvider(&value.Record))
		}
	}
	return out, nil
}

func (s *mixedHTTPProviders) ListSchedulableByGroupIDAndPlatform(ctx context.Context, id int64, p string) ([]gatewayadapter.ExecutionProvider, error) {
	return s.ListSchedulableByGroupIDAndPlatforms(ctx, id, []string{p})
}
func (s *mixedHTTPProviders) SetError(context.Context, int64, string) error { return nil }
func (s *mixedHTTPProviders) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}
func (s *mixedHTTPProviders) UpdateLastUsed(context.Context, int64) error { return nil }
func (s *mixedHTTPProviders) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}

// mixedHTTPTransport 真正经过本地 HTTP server，记录实际选择提供商和上游端点。
type mixedHTTPTransport struct {
	mu        sync.Mutex
	providers []int64
}

func (s *mixedHTTPTransport) Do(req *http.Request, _ string, id int64, _ int) (*http.Response, error) {
	s.mu.Lock()
	s.providers = append(s.providers, id)
	s.mu.Unlock()
	return http.DefaultClient.Do(req)
}

func (s *mixedHTTPTransport) DoWithTLS(req *http.Request, p string, id int64, n int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(req, p, id, n)
}

func (s *mixedHTTPTransport) calls() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.providers)
}

type mixedHTTPNoSearch struct{}

func (mixedHTTPNoSearch) Current() searchtools.Searcher { return nil }

func mixedUpstreamResponse(w http.ResponseWriter, r *http.Request, failAnthropic bool) {
	body, _ := io.ReadAll(r.Body)
	if failAnthropic && strings.Contains(r.URL.Path, "/messages") {
		w.WriteHeader(502)
		_, _ = io.WriteString(w, `{"error":{"type":"overloaded_error","message":"retry another provider"}}`)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if strings.Contains(r.URL.Path, "/messages") {
		if gjson.GetBytes(body, "stream").Bool() {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg-mixed\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-sonnet-4-5-20250929\",\"content\":[],\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}\n\nevent: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok-mixed\"}}\n\nevent: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":3}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
		} else {
			_, _ = io.WriteString(w, `{"id":"msg-mixed","type":"message","role":"assistant","model":"claude-sonnet-4-5-20250929","content":[{"type":"text","text":"ok-mixed"}],"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":3}}`)
		}
	} else if strings.Contains(r.URL.Path, "models/") {
		response := `{"candidates":[{"content":{"role":"model","parts":[{"text":"ok-mixed"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":3,"totalTokenCount":13}}`
		if strings.Contains(r.URL.Path, "streamGenerateContent") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "data: %s\n\n", response)
		} else {
			_, _ = io.WriteString(w, response)
		}
	} else if strings.Contains(r.URL.Path, "chat/completions") {
		_, _ = io.WriteString(w, `{"id":"chat-mixed","object":"chat.completion","model":"gpt-5.4","choices":[{"index":0,"message":{"role":"assistant","content":"ok-mixed"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}}`)
	} else {
		response := `{"id":"resp-mixed","object":"response","status":"completed","model":"gpt-5.4","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok-mixed"}]}],"usage":{"input_tokens":10,"output_tokens":3,"total_tokens":13}}`
		if gjson.GetBytes(body, "stream").Bool() {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp-mixed\",\"model\":\"gpt-5.4\"}}\n\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"ok-mixed\"}\n\ndata: {\"type\":\"response.completed\",\"response\":%s}\n\n", response)
		} else {
			_, _ = io.WriteString(w, response)
		}
	}
}

// TestMixedGroupTextHTTP 用实际选择、协议转换及完成器覆盖三种入口与跨平台故障转移。
func TestMixedGroupTextHTTP(t *testing.T) {
	for _, path := range []string{"/v1/messages", "/v1/responses", "/v1/chat/completions"} {
		for _, platform := range []string{"anthropic", "openai", "gemini", "failover"} {
			t.Run(path+"/"+platform, func(t *testing.T) {
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mixedUpstreamResponse(w, r, platform == "failover") }))
				defer upstream.Close()
				groupID := int64(42)
				group := &routing.Group{ID: groupID, Hydrated: true, Name: "mixed", Status: "active", RateMultiplier: 1, AllowedProtocols: []protocol.ProtocolID{protocol.ProtocolAnthropicMessages, protocol.ProtocolOpenAIResponses, protocol.ProtocolOpenAIChatCompletions}}
				cfg := &config.Config{}
				cfg.Default.RateMultiplier = 1
				cfg.Security.URLAllowlist.Enabled = false
				cfg.Security.URLAllowlist.AllowInsecureHTTP = true
				cfg.Gateway.MaxProviderSwitches = 2
				providers := &mixedHTTPProviders{}
				for index, p := range []string{"anthropic", "openai", "gemini"} {
					record := &provider.Record{ID: int64(index + 1), Name: p, Platform: p, Type: "apikey", Status: "active", Schedulable: true, Priority: index + 1, Concurrency: 1, GroupIDs: []int64{groupID}, Credentials: map[string]any{"api_key": "test-key", "base_url": upstream.URL}}
					if p == "openai" && platform == "failover" {
						record.Credentials["model_mapping"] = map[string]any{"claude-sonnet-4-5-20250929": "gpt-5.4"}
					}
					providers.values = append(providers.values, *gatewayadapter.NewExecutionProvider(record))
				}
				transport := &mixedHTTPTransport{}
				source, choices, credentials := newOpenAIExecutionAndSelectionFixture(providers, nil, cfg, nil, nil, nil, transport, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
				logs := &gatewaytestkit.UsageLogStore{Inserted: true}
				funds := &gatewaytestkit.SettlementStore{}
				recording := gatewaytestkit.NewRecording(logs, funds, nil, false)
				recording.Options.DefaultMultiplier = 1
				source.Recorder = recording.Core(nil, true)
				native := &gatewayhttp.UnifiedTextExecutor{OpenAI: source.Responses, Anthropic: gatewayhttp.NewMessagesExecutor(messageforward.NewRuntime(messageforward.Dependencies{Credentials: &provider.MessageCredentialSource{}, Transport: transport, Deferred: &provider.DeferredService{}, Search: searchtools.NewEmulator(mixedHTTPNoSearch{}, nil, nil, nil, nil, nil)}, messageforward.Options{Configured: true, AllowInsecureHTTP: true, ResponseReadLimit: 1 << 20}), nil), Gemini: &gatewayhttp.GeminiExecutor{Runtime: &googleforward.Gemini{Transport: transport, Options: googleforward.Options{Configured: true, AllowInsecureHTTP: true, ResponseReadLimit: 1 << 20}}}}
				eligibility := newBillingEligibilityFixture(cfg)
				t.Cleanup(eligibility.Stop)
				resources := provideOpenAIHTTPResources(scheduler.NewConcurrencyService(nil), cfg)
				endpoints := newGatewayHTTPEndpoints(gatewayHTTPFixtureInput{Source: source, Native: native, Credentials: credentials, Choices: choices, Config: cfg, Funding: newFundingAdmissionFixture(eligibility, cfg), Keys: keytestkit.NewService(nil, nil, nil, nil, nil, nil, cfg), Concurrency: resources.Concurrency, MaxSwitches: 2})
				user := &identity.User{ID: 7, Balance: 100, Status: "active"}
				key := &apikey.APIKey{ID: 8, UserID: 7, User: user, GroupID: &groupID, Group: group}
				router := gin.New()
				router.Use(func(c *gin.Context) {
					c.Set(string(keyhttp.ContextKeyAPIKey), key)
					c.Set("gateway_effective_key", key)
					authctx.SetPrincipal(c, identity.Principal{UserID: 7}, 0, "")
					sourceProtocol := protocol.ProtocolAnthropicMessages
					if path == "/v1/responses" {
						sourceProtocol = protocol.ProtocolOpenAIResponses
					}
					if path == "/v1/chat/completions" {
						sourceProtocol = protocol.ProtocolOpenAIChatCompletions
					}
					c.Request = c.Request.WithContext(requeststate.WithClientProtocol(requeststate.WithGroup(c.Request.Context(), group), sourceProtocol))
				})
				router.POST("/v1/messages", endpoints.Messages)
				router.POST("/v1/responses", endpoints.Responses)
				router.POST("/v1/chat/completions", endpoints.ChatCompletions)
				model := map[string]string{"anthropic": "claude-sonnet-4-5-20250929", "openai": "gpt-5.4", "gemini": "gemini-2.5-flash", "failover": "claude-sonnet-4-5-20250929"}[platform]
				body := fmt.Sprintf(`{"model":%q,"max_tokens":20,"messages":[{"role":"user","content":"hello"}],"stream":false}`, model)
				if path == "/v1/responses" {
					body = fmt.Sprintf(`{"model":%q,"input":"hello","stream":false}`, model)
				}
				response := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(response, request)
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
				require.Contains(t, response.Body.String(), "ok-mixed")
				expected := map[string]int64{"anthropic": 1, "openai": 2, "gemini": 3, "failover": 2}[platform]
				calls := transport.calls()
				require.NotEmpty(t, calls)
				require.Equal(t, expected, calls[len(calls)-1])
				if platform == "failover" {
					require.Equal(t, []int64{1, 2}, calls)
				} else {
					require.Len(t, calls, 1)
				}
				require.Equal(t, 1, funds.Calls, "一次成功只能提交一次资金扣费")
				require.Equal(t, 1, logs.Calls)
				require.Equal(t, expected, logs.LastLog.ProviderID)
				require.Equal(t, map[int64]string{1: "anthropic", 2: "openai", 3: "gemini"}[expected], logs.LastLog.Platform)
			})
		}
	}
}
