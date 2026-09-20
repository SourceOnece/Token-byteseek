//go:build unit

package service

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setupTicketProxyTest(t *testing.T, mode string, attempts int) (*CodexTicketService, *ticketCacheStub, *Account) {
	t.Helper()
	s, cache, _ := newTicketTestService()
	enableTicketTest(t, s)
	list := []CodexTicketProxyUpdate{{ID: "legacy", Name: "A"}, {ID: "11111111-1111-4111-8111-111111111111", Name: "B", URL: "http://user:synthetic-password@proxy-b.invalid:1080"}}
	fixed := "legacy"
	retry, interval := 1, 6
	_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{Proxies: &list, SelectionMode: &mode, FixedProxyID: &fixed, MaxAttempts: &attempts, RetryIntervalSeconds: &retry, ProbeIntervalSeconds: &interval})
	require.NoError(t, err)
	a := ticketAccount()
	s.gateway = &OpenAIGatewayService{accountRepo: &ticketAccountStub{accounts: []Account{a}}}
	require.NoError(t, configureTicketTestAccount(t, s, CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{MaxAttempts: &attempts, RetryIntervalSeconds: &retry, ProbeIntervalSeconds: &interval}}))
	return s, cache, &a
}

// 网络替身捕获实际代理和载荷，只返回合成响应，不访问 OpenAI 或真实代理。
type ticketSequenceUpstream struct {
	HTTPUpstream
	proxies   []string
	bodies    [][]byte
	responses []*http.Response
	headers   []http.Header
}

func (u *ticketSequenceUpstream) Do(req *http.Request, proxy string, _ int64, _ int) (*http.Response, error) {
	u.proxies = append(u.proxies, proxy)
	body, _ := io.ReadAll(req.Body)
	u.bodies = append(u.bodies, body)
	u.headers = append(u.headers, req.Header.Clone())
	r := u.responses[0]
	if len(u.responses) > 1 {
		u.responses = u.responses[1:]
	}
	return r, nil
}
func ticketResponseForProxy(status, length int, body string) *http.Response {
	h := http.Header{"Content-Type": {"text/event-stream"}}
	if length > 0 {
		h.Set(openAICodexTurnStateHeader, "gAAAAA"+strings.Repeat("x", length-6))
	}
	return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(body))}
}

func TestCodexTicketProxySettingsCompatibilityAndPrivacy(t *testing.T) {
	s, cache, r := newTicketTestService()
	enableTicketTest(t, s)
	v, err := s.View(context.Background())
	require.NoError(t, err)
	require.Equal(t, "fixed", v.SelectionMode)
	require.Equal(t, 6, v.ProbeIntervalSeconds)
	require.Equal(t, 1, v.MaxAttempts)
	require.Equal(t, "legacy", v.FixedProxyID)
	list := []CodexTicketProxyUpdate{{ID: "legacy", Name: "Renamed"}, {Name: "New", URL: "https://user:synthetic-password@proxy.invalid:443"}}
	v, err = s.Update(context.Background(), CodexTicketSettingsUpdate{Proxies: &list, Revision: &v.Revision})
	require.NoError(t, err)
	require.Len(t, v.Proxies, 2)
	raw, _ := json.Marshal(v)
	require.NotContains(t, string(raw), "synthetic-password")
	require.NotContains(t, string(raw), "proxy.invalid")
	require.NotContains(t, string(raw), "cipher")
	stored, _ := r.GetValue(context.Background(), codexTicketSettingsKey)
	require.NotContains(t, stored, "synthetic-password")
	_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{ClearProxy: true})
	require.Error(t, err, "旧界面不能抹掉多代理")
	stale := "old"
	_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{Revision: &stale})
	require.Error(t, err)
	mode := "rotate"
	_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{SelectionMode: &mode})
	require.NoError(t, err)
	before := cache.gets
	no := false
	_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{Enabled: &no})
	require.NoError(t, err)
	s.Apply(context.Background(), &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth}, "gpt-6-astra", http.Header{"Authorization": {"Bearer synthetic"}})
	require.Equal(t, before, cache.gets)
}

func TestCodexTicketProxyValidationAndSelection(t *testing.T) {
	s, _, _ := setupTicketProxyTest(t, "rotate", 3)
	cfg := s.config.Load()
	list := cfg.proxies()
	p, ok := selectCodexTicketProxy(cfg, list[0].ID, false)
	require.True(t, ok)
	require.Equal(t, list[0].ID, p.ID, "成功不轮换")
	p, ok = selectCodexTicketProxy(cfg, list[0].ID, true)
	require.True(t, ok)
	require.Equal(t, list[1].ID, p.ID)
	p, _ = selectCodexTicketProxy(cfg, list[1].ID, true)
	require.Equal(t, list[0].ID, p.ID)
	copy := *cfg
	copy.SelectionMode = "fixed"
	copy.FixedProxyID = list[1].ID
	p, _ = selectCodexTicketProxy(&copy, list[0].ID, true)
	require.Equal(t, list[1].ID, p.ID)
	for _, bad := range []CodexTicketSettingsUpdate{
		{MaxAttempts: ticketInt(-1)}, {TargetLength: ticketInt(5)}, {ProbeIntervalSeconds: ticketInt(5)}, {RetryIntervalSeconds: ticketInt(0)},
		{Proxies: &[]CodexTicketProxyUpdate{{ID: "missing", Name: "X"}}}, {Proxies: &[]CodexTicketProxyUpdate{{ID: "legacy", Name: "A"}, {ID: "legacy", Name: "A"}}},
		{Proxies: &[]CodexTicketProxyUpdate{{Name: "", URL: "http://proxy"}}}, {Proxies: &[]CodexTicketProxyUpdate{}},
	} {
		_, err := s.Update(context.Background(), bad)
		require.Error(t, err)
	}
}
func ticketInt(value int) *int { return &value }

func TestCodexTicketProxyRetryOnlyAfterMissAndStopOnSuccess(t *testing.T) {
	s, _, a := setupTicketProxyTest(t, "rotate", 3)
	u := &ticketSequenceUpstream{responses: []*http.Response{ticketResponseForProxy(200, 312, ""), ticketResponseForProxy(200, 292, "")}}
	s.gateway.httpUpstream = u
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	require.Len(t, u.proxies, 2)
	require.Contains(t, u.proxies[0], "proxy.example")
	require.Contains(t, u.proxies[1], "proxy-b.invalid")
	require.Contains(t, string(u.bodies[0]), `"content":[{"text":"ping","type":"input_text"}]`)
	require.Equal(t, "responses=experimental", u.headers[0].Get("OpenAI-Beta"))
	require.GreaterOrEqual(t, CompareVersions(u.headers[0].Get("version"), "0.153.4"), 0)
	status, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	got := status.Items[0].Models[0]
	require.Equal(t, "ready", got.State)
	require.Equal(t, "B", got.Diagnostic.ProxyName)
	require.Equal(t, 2, got.Diagnostic.Attempt)
	require.Equal(t, 292, got.Diagnostic.HeaderLength)
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	require.Len(t, u.proxies, 2, "有效且未近过期的票不重试")
	require.True(t, a.Schedulable)
}

func TestCodexTicketFixedRetryAndAuthLimitStop(t *testing.T) {
	s, _, a := setupTicketProxyTest(t, "fixed", 3)
	u := &ticketSequenceUpstream{responses: []*http.Response{ticketResponseForProxy(200, 312, ""), ticketResponseForProxy(200, 292, "")}}
	s.gateway.httpUpstream = u
	s.probe(context.Background(), s.config.Load(), a, "gpt-5.6-sol")
	require.Len(t, u.proxies, 2)
	require.Equal(t, u.proxies[0], u.proxies[1])
	require.GreaterOrEqual(t, CompareVersions(u.headers[0].Get("version"), "0.146.0"), 0)
	for _, code := range []int{401, 403, 429} {
		s, _, a = setupTicketProxyTest(t, "rotate", 3)
		u = &ticketSequenceUpstream{responses: []*http.Response{ticketResponseForProxy(code, 0, "")}}
		s.gateway.httpUpstream = u
		s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
		require.Len(t, u.proxies, 1)
	}
}

func TestCodexTicketFailureDiagnosticsAndNextCycleProxy(t *testing.T) {
	s, cache, a := setupTicketProxyTest(t, "rotate", 1)
	u := &ticketSequenceUpstream{responses: []*http.Response{ticketResponseForProxy(200, 0, "data: {\"type\":\"error\",\"error\":{\"code\":\"server_is_overloaded\",\"message\":\"secret-canary\"}}\n\n"), ticketResponseForProxy(200, 292, "")}}
	s.gateway.httpUpstream = u
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	status, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	row := status.Items[0].Models[0]
	require.Equal(t, "failed", row.State)
	require.Equal(t, "overloaded", row.Diagnostic.ErrorKind)
	require.Equal(t, 200, row.Diagnostic.HTTPStatus)
	require.False(t, row.Diagnostic.HeaderPresent)
	raw, _ := json.Marshal(status)
	require.NotContains(t, string(raw), "secret-canary")
	cache.mu.Lock()
	cache.claims = map[string]bool{}
	cache.mu.Unlock()
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	require.Contains(t, u.proxies[1], "proxy-b.invalid", "下一轮从上次失败后的代理继续")
}

func TestCodexTicketRetryCancelledDuringDelay(t *testing.T) {
	s, _, a := setupTicketProxyTest(t, "rotate", 3)
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(50*time.Millisecond, cancel)
	defer timer.Stop()
	defer cancel()
	u := &ticketSequenceUpstream{responses: []*http.Response{ticketResponseForProxy(200, 312, "")}}
	s.gateway.httpUpstream = u
	s.probe(ctx, s.config.Load(), a, "gpt-6-astra")
	require.Len(t, u.proxies, 1, "取消等待后不能换代理发下一次")
}

func TestCodexTicketProxyRetryIsBounded(t *testing.T) {
	s, _, a := setupTicketProxyTest(t, "rotate", 3)
	u := &ticketSequenceUpstream{responses: []*http.Response{ticketResponseForProxy(200, 312, "")}}
	s.gateway.httpUpstream = u
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	require.Len(t, u.proxies, 3)
	require.Equal(t, u.proxies[0], u.proxies[2])
	require.NotEqual(t, u.proxies[0], u.proxies[1])
	status, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, "missing", status.Items[0].Models[0].State)
	require.Equal(t, 312, status.Items[0].Models[0].Diagnostic.HeaderLength)
}

func TestCodexTicketFailureDiagnosticReadsBoundedData(t *testing.T) {
	for _, test := range []struct{ body, kind string }{
		{"{\n\"error\": {\"code\": \"insufficient_quota\", \"message\": \"secret\"}\n}", "quota"},
		{"data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"code\":\"rate_limit_exceeded\"}}}\n\n", "rate_limit"},
		{"data: {\"error\":{\"message\":\"Our servers are currently overloaded. Please try again later.\"}}\n\n", "overloaded"},
		{"data: {\"type\":\"error\",\"code\":\"server_error\",\"message\":\"Our servers are currently overloaded. Please try again later.\"}\n\n", "overloaded"},
	} {
		d := &CodexTicketDiagnostic{}
		readTicketFailureDiagnostic(io.NopCloser(strings.NewReader(test.body)), d, func() {})
		require.Equal(t, test.kind, d.ErrorKind)
		raw, _ := json.Marshal(d)
		require.NotContains(t, string(raw), "secret")
	}
	d := &CodexTicketDiagnostic{}
	readTicketFailureDiagnostic(io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\"}\n")), d, func() {})
	require.True(t, d.CompletionSeen)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	response, err := server.Client().Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	start := time.Now()
	readTicketFailureDiagnostic(response.Body, &CodexTicketDiagnostic{}, cancel)
	require.Less(t, time.Since(start), 5*time.Second)
	require.Error(t, ctx.Err(), "无数据流应取消单次请求，而不是一直等 SSE")
}

func TestCodexTicketRetryAfterDoesNotSwitchOrProbeEarly(t *testing.T) {
	s, cache, a := setupTicketProxyTest(t, "rotate", 3)
	response := ticketResponseForProxy(503, 0, "")
	response.Header.Set("Retry-After", "120")
	u := &ticketSequenceUpstream{responses: []*http.Response{response}}
	s.gateway.httpUpstream = u
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	require.Len(t, u.proxies, 1)
	status, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.NotNil(t, status.Items[0].Models[0].Diagnostic.RetryNotBefore)
	cache.mu.Lock()
	cache.claims = map[string]bool{}
	cache.mu.Unlock()
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	require.Len(t, u.proxies, 1)
	require.Nil(t, codexTicketRetryNotBefore("", 200, ""))
	require.NotNil(t, codexTicketRetryNotBefore("", 429, ""))
}
