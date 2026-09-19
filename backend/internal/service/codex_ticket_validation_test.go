//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func ticketCompletedBody(model string) string {
	return `data: {"type":"response.completed","response":{"status":"completed","model":"` + model + `"}}` + "\n\n"
}

func TestCodexTicketCompleteModelValidation(t *testing.T) {
	for _, tc := range []struct {
		name, body, model, reason string
		complete                  bool
	}{
		{"sse", ticketCompletedBody("gpt-6-astra"), "gpt-6-astra", "", true},
		{"mismatch", ticketCompletedBody("gpt-5.6-luna"), "gpt-5.6-luna", "model_mismatch", true},
		{"json", `{"object":"response","status":"completed","model":"gpt-6-astra"}`, "gpt-6-astra", "", true},
		{"incomplete", `data: {"type":"response.incomplete"}` + "\n\n", "", "incomplete_response", false},
		{"missing_model", `data: {"type":"response.completed","response":{"status":"completed"}}` + "\n\n", "", "incomplete_response", false},
		{"delta_only", `data: {"type":"response.output_text.delta","delta":"pong"}` + "\n\n", "", "incomplete_response", false},
		{"done_only", "data: [DONE]\n\n", "", "incomplete_response", false},
		{"truncated", `data: {"type":"response.completed"`, "", "incomplete_response", false},
		{"large", strings.Repeat("x", (2<<20)+8), "", "incomplete_response", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := parseTicketCompletion(strings.NewReader(tc.body), "gpt-6-astra")
			require.Equal(t, tc.model, got.Model)
			require.Equal(t, tc.reason, got.Reason)
			require.Equal(t, tc.complete, got.Complete)
		})
	}
}

type ticketVerifiedCall struct {
	proxy, state string
	verify       bool
}
type ticketVerifiedUpstream struct {
	HTTPUpstream
	responses []*http.Response
	calls     []ticketVerifiedCall
	before    func(int)
}

func (u *ticketVerifiedUpstream) call(req *http.Request, proxy string, verify bool) (*http.Response, error) {
	u.calls = append(u.calls, ticketVerifiedCall{proxy: proxy, state: req.Header.Get(openAICodexTurnStateHeader), verify: verify})
	if u.before != nil {
		u.before(len(u.calls))
	}
	if len(u.calls) > len(u.responses) {
		return nil, nil
	}
	return u.responses[len(u.calls)-1], nil
}
func (u *ticketVerifiedUpstream) Do(req *http.Request, proxy string, _ int64, _ int) (*http.Response, error) {
	return u.call(req, proxy, false)
}
func (u *ticketVerifiedUpstream) DoWithTLS(req *http.Request, proxy string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.call(req, proxy, true)
}
func verifiedResponse(code, length int, model string) *http.Response {
	h := http.Header{"Content-Type": []string{"text/event-stream"}}
	if length > 0 {
		h.Set(openAICodexTurnStateHeader, "gAAAAA"+strings.Repeat("x", length-6))
	}
	return &http.Response{StatusCode: code, Header: h, Body: io.NopCloser(strings.NewReader(ticketCompletedBody(model)))}
}

func setupVerifiedTicketTest(t *testing.T, enabled bool) (*CodexTicketService, *ticketSchedulingRepo, *ticketVerifiedUpstream) {
	t.Helper()
	s, base, _ := setupTicketManualTest(t, 292)
	proxyID := int64(17)
	base.accounts[0].ProxyID = &proxyID
	base.accounts[0].Proxy = &Proxy{ID: 17, Protocol: "http", Host: "business.example", Port: 8080, Status: StatusActive}
	r := &ticketSchedulingRepo{ticketHistoryStub: base}
	u := &ticketVerifiedUpstream{}
	s.gateway.accountRepo = r
	s.gateway.httpUpstream = u
	models := []string{"gpt-6-astra"}
	signal := 312
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{VerifiedFlow: &enabled, Rules: &CodexTicketRulesPatch{Models: &models, DegradedSignalLength: &signal}}})
	require.NoError(t, err)
	return s, r, u
}

func TestCodexTicketDualRoutePublicationMatrix(t *testing.T) {
	for _, tc := range []struct {
		name                                     string
		firstModel, secondModel                  string
		firstLength, secondLength, status, calls int
		ready                                    bool
	}{
		{"pass", "gpt-6-astra", "gpt-6-astra", 292, 0, 200, 2, true},
		{"candidate_mismatch", "gpt-5.6-luna", "gpt-6-astra", 292, 0, 200, 1, false},
		{"candidate_length", "gpt-6-astra", "gpt-6-astra", 356, 0, 200, 1, false},
		{"verify_mismatch", "gpt-6-astra", "gpt-5.6-luna", 292, 0, 200, 2, false},
		{"verify_signal", "gpt-6-astra", "gpt-6-astra", 292, 312, 200, 2, false},
		{"verify_429", "gpt-6-astra", "gpt-6-astra", 292, 0, 429, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, r, u := setupVerifiedTicketTest(t, true)
			u.responses = []*http.Response{verifiedResponse(200, tc.firstLength, tc.firstModel), verifiedResponse(tc.status, tc.secondLength, tc.secondModel)}
			a, _ := r.GetByID(context.Background(), 1)
			cfg := s.enabledAccountConfig(1)
			key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
			s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
			require.Len(t, u.calls, tc.calls)
			value, ok := s.lookup(context.Background(), cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
			require.Equal(t, tc.ready, ok)
			if tc.ready {
				require.True(t, value.Verified)
				require.Equal(t, ticketBusinessFingerprint(a), value.BusinessFingerprint)
				require.Empty(t, u.calls[0].state)
				require.NotEqual(t, u.calls[0].proxy, u.calls[1].proxy)
				require.Equal(t, a.Proxy.URL(), u.calls[1].proxy)
				require.True(t, u.calls[1].verify)
				require.Len(t, u.calls[1].state, 292)
			}
			raw, _ := s.cache.Get(context.Background(), "latest:"+key)
			require.NotContains(t, raw, "gAAAAA")
			var latest CodexTicketLatest
			require.NoError(t, json.Unmarshal([]byte(raw), &latest))
			require.Len(t, latest.Diagnostic.Stages, tc.calls)
		})
	}
}

func TestCodexTicketVerifiedFlowOffAndBusinessProxyRequired(t *testing.T) {
	s, r, u := setupVerifiedTicketTest(t, false)
	u.responses = []*http.Response{verifiedResponse(200, 292, "gpt-5.6-luna")}
	a, _ := r.GetByID(context.Background(), 1)
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	require.Len(t, u.calls, 1, "关闭新模式沿旧长度判断，不添加验证调用")
	_, ok := s.lookup(context.Background(), s.enabledAccountConfig(1), a, "gpt-6-astra", a.GetOpenAIAccessToken())
	require.True(t, ok)
	s, r, u = setupVerifiedTicketTest(t, true)
	r.accounts[0].ProxyID = nil
	r.accounts[0].Proxy = nil
	a, _ = r.GetByID(context.Background(), 1)
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	require.Empty(t, u.calls, "无业务代理不能偷偷直连或先消耗采集额度")
}

func TestCodexTicketVerifiedRenewalRetainsUsableOldTicketAndBindsProxy(t *testing.T) {
	s, r, u := setupVerifiedTicketTest(t, true)
	a, _ := r.GetByID(context.Background(), 1)
	cfg := s.enabledAccountConfig(1)
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	old := codexTicketValue{State: "gAAAAA" + strings.Repeat("y", 286), Verified: true, BusinessFingerprint: ticketBusinessFingerprint(a), ExpiresAt: time.Now().Add(5 * time.Minute)}
	raw, _ := json.Marshal(old)
	encrypted, _ := s.cipher.Encrypt(string(raw))
	require.NoError(t, s.cache.Set(context.Background(), key, encrypted, time.Hour))
	u.responses = []*http.Response{verifiedResponse(200, 292, "gpt-5.6-luna")}
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	after, _ := s.cache.Get(context.Background(), key)
	require.Equal(t, encrypted, after)
	cooldown, _ := json.Marshal(map[string]any{"failures": 3, "until": time.Now().Add(time.Minute).UnixMilli()})
	_ = s.cache.Set(context.Background(), "collection:"+key, string(cooldown), time.Hour)
	_, ok := s.lookup(context.Background(), cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	require.True(t, ok)
	copy := *a
	proxyCopy := *a.Proxy
	proxyCopy.Host = "new-business.example"
	copy.Proxy = &proxyCopy
	require.NotEqual(t, key, codexTicketKey(cfg, &copy, "gpt-6-astra", a.GetOpenAIAccessToken()))
	_, ok = s.lookup(context.Background(), cfg, &copy, "gpt-6-astra", a.GetOpenAIAccessToken())
	require.False(t, ok)
	// 实际发出请求前还需拒绝长会话捕获的旧账号出口，即使旧key的票仍未到期。
	r.accounts[0].Proxy = &proxyCopy
	h := http.Header{"Authorization": []string{"Bearer " + a.GetOpenAIAccessToken()}}
	require.ErrorIs(t, s.Apply(context.Background(), a, "gpt-6-astra", h), ErrCodexTicketUnavailable)
}

func TestCodexTicketVerifiedWSBridgeDecision(t *testing.T) {
	s, r, _ := setupVerifiedTicketTest(t, true)
	s.gateway.codexTickets.Store(s)
	a, _ := r.GetByID(context.Background(), 1)
	allowed := OpenAIWSProtocolDecision{Transport: OpenAIUpstreamTransportResponsesWebsocketV2}
	require.True(t, s.gateway.shouldBridgeVerifiedTicketAccount(a, allowed))
	for _, reason := range []string{"global_disabled", "account_mode_off", "account_force_http"} {
		require.False(t, s.gateway.shouldBridgeVerifiedTicketAccount(a, OpenAIWSProtocolDecision{Transport: OpenAIUpstreamTransportHTTPSSE, Reason: reason}))
	}
	no := false
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{VerifiedFlow: &no}})
	require.NoError(t, err)
	require.False(t, s.gateway.shouldBridgeVerifiedTicketAccount(a, allowed))
}

// 走真实桥接回合函数，验证每轮读新票、记录真实模型，以及不重放异常回答。
func TestCodexTicketVerifiedWSBridgeReadsEachTicketAndObservesResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, r, u := setupVerifiedTicketTest(t, true)
	s.cache = &ticketWatchdogCacheStub{s.cache.(*ticketCacheStub)}
	s.gateway.cfg = &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	s.gateway.codexTickets.Store(s)
	a, _ := r.GetByID(context.Background(), 1)
	cfg := s.enabledAccountConfig(1)
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	seed := func(letter string) string {
		state := "gAAAAA" + strings.Repeat(letter, 286)
		raw, _ := json.Marshal(codexTicketValue{State: state, Verified: true, BusinessFingerprint: ticketBusinessFingerprint(a), ExpiresAt: time.Now().Add(time.Hour)})
		encoded, _ := s.cipher.Encrypt(string(raw))
		require.NoError(t, s.cache.Set(context.Background(), key, encoded, time.Hour))
		return state
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/responses", nil)
	payload := []byte(`{"type":"response.create","model":"gpt-6-astra","input":"hi"}`)
	u.responses = []*http.Response{verifiedResponse(200, 0, "gpt-6-astra"), verifiedResponse(200, 0, "gpt-5.6-luna")}
	first := seed("a")
	result, err := s.gateway.proxyOpenAIWSHTTPBridgeTurn(context.Background(), c, a, a.GetOpenAIAccessToken(), payload, len(payload), "gpt-6-astra", "", "", "", "", 1, func([]byte) error { return nil })
	require.NoError(t, err)
	require.Equal(t, first, u.calls[0].state)
	require.Equal(t, "gpt-6-astra", result.UpstreamResponseModel)
	second := seed("b")
	result, err = s.gateway.proxyOpenAIWSHTTPBridgeTurn(context.Background(), c, a, a.GetOpenAIAccessToken(), payload, len(payload), "gpt-6-astra", "", "", "", "", 2, func([]byte) error { return nil })
	require.NoError(t, err)
	require.Equal(t, second, u.calls[1].state)
	require.Equal(t, "gpt-5.6-luna", result.UpstreamResponseModel)
	require.Len(t, u.calls, 2, "异常业务回答不能自动重放")
	value, err := s.cache.Get(context.Background(), key)
	require.NoError(t, err)
	require.Empty(t, value)
	require.True(t, s.watchdogWake.Load())
	// 同一客户端下一轮缺少模型声明时，日志必须为空，不能继承刚才的Luna。
	seed("c")
	u.responses = append(u.responses, verifiedResponse(200, 0, ""))
	result, err = s.gateway.proxyOpenAIWSHTTPBridgeTurn(context.Background(), c, a, a.GetOpenAIAccessToken(), payload, len(payload), "gpt-6-astra", "", "", "", "", 3, func([]byte) error { return nil })
	require.NoError(t, err)
	require.Empty(t, result.UpstreamResponseModel)
	require.Len(t, u.calls, 3)
}
