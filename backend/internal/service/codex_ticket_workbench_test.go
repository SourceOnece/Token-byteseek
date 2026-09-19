//go:build unit

package service

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCodexTicketAccountRulesAndGlobalSwitchOnly(t *testing.T) {
	s, _, _ := newTicketTestService()
	enableTicketTest(t, s)
	a, b := ticketAccount(), ticketAccount()
	b.ID = 2
	s.gateway = &OpenAIGatewayService{accountRepo: &ticketAccountStub{accounts: []Account{a, b}}}
	ctx := context.Background()
	zero, cache, renew, parallel := 0, 120, 15, 1
	models := []string{"custom-model"}
	out, err := s.UpdateAccountSettings(ctx, CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{Models: &models, MaxAttempts: &zero, CacheMinutes: &cache, RefreshBeforeMinutes: &renew, Concurrency: &parallel}}})
	require.NoError(t, err)
	require.Zero(t, out[0].Rules.MaxAttempts)
	one := s.enabledAccountConfig(1)
	two := s.enabledAccountConfig(2)
	require.Zero(t, one.attempts())
	require.Equal(t, 2*time.Hour, one.ticketTTL())
	require.Equal(t, 15*time.Minute, one.refreshBefore())
	require.Equal(t, 1, one.collectionConcurrency())
	require.Equal(t, models, one.models())
	require.Equal(t, time.Hour, two.ticketTTL())
	require.Equal(t, []string{"gpt-6-astra", "gpt-5.6-sol"}, two.models())
	falseValue := false
	_, err = s.Update(ctx, CodexTicketSettingsUpdate{Enabled: &falseValue})
	require.NoError(t, err)
	require.Nil(t, s.enabledAccountConfig(1))
	got, err := s.AccountSettings(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 120, got.Rules.CacheMinutes)
}

func TestCodexTicketRulesRejectUnsafeBounds(t *testing.T) {
	r := ticketRulesFromConfig(&codexTicketConfig{})
	for _, p := range []CodexTicketRulesPatch{{MaxAttempts: ticketInt(-1)}, {Concurrency: ticketInt(0)}, {Concurrency: ticketInt(5)}, {CacheMinutes: ticketInt(0)}, {RefreshBeforeMinutes: ticketInt(60)}, {DegradedSignalLength: ticketInt(1)}, {CooldownSeconds: ticketInt(0)}} {
		_, err := applyTicketRulesPatch(r, &p)
		require.Error(t, err)
	}
}

type ticketProviderTransport func(*http.Request) (*http.Response, error)

func (f ticketProviderTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCodexTicketExtractionEveryAttemptAndSecretSafety(t *testing.T) {
	s, _, a := setupTicketProxyTest(t, "fixed", 3)
	endpoint := "https://provider.example/v1/gen?user=synthetic&pass=secret-example&country=tw&amout=1"
	input := CodexTicketProxyPolicyUpdate{Mode: "dynamic", DynamicSource: "api", ExtractionURL: &endpoint, ProxyProtocol: "http"}
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{ProxyPolicy: &input}})
	require.NoError(t, err)
	requests := 0
	s.proxyProviderClient = &http.Client{Transport: ticketProviderTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		require.Equal(t, "secret-example", r.URL.Query().Get("pass"))
		require.Empty(t, r.Header.Get("Authorization"))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"proxies":["8.8.8.8:8080:client:opaque-pass"]}`)), Header: http.Header{}}, nil
	})}
	u := &ticketSequenceUpstream{responses: []*http.Response{ticketResponseForProxy(200, 312, ""), ticketResponseForProxy(200, 292, "")}}
	s.gateway.httpUpstream = u
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	require.Equal(t, 2, requests)
	require.Len(t, u.proxies, 2)
	for _, proxy := range u.proxies {
		p, e := url.Parse(proxy)
		require.NoError(t, e)
		password, _ := p.User.Password()
		require.Equal(t, "opaque-pass", password)
	}
	view, err := s.AccountSettings(context.Background(), 1)
	require.NoError(t, err)
	raw, _ := json.Marshal(view)
	for _, secret := range []string{endpoint, "opaque-pass", "secret-example"} {
		require.NotContains(t, string(raw), secret)
	}
	_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{ProxyPolicy: &input})
	require.NoError(t, err)
	no := false
	_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{Enabled: &no})
	require.NoError(t, err, "API取号配置不能妨碍关闭总闸")
}

func TestCodexTicketProviderParsingAndPublicTargets(t *testing.T) {
	for _, raw := range []string{`{"proxies":["1.1.1.1:8080:user:pa:ss"]}`, "1.1.1.1:8080:user:password\n", "http://user:p%40ss@1.1.1.1:8080"} {
		proxy, err := parseTicketProviderResponse([]byte(raw), "http")
		require.NoError(t, err)
		require.Contains(t, proxy, "1.1.1.1:8080")
	}
	for _, raw := range []string{`{"error":"do something else"}`, "not a proxy", "1.1.1.1:99999:u:p", "ftp://1.1.1.1:21"} {
		_, err := parseTicketProviderResponse([]byte(raw), "http")
		require.Error(t, err)
	}
	for _, raw := range []string{"http://provider.invalid", "https://127.0.0.1/a", "https://user:pass@provider.invalid", "https://provider.invalid/#x"} {
		require.Error(t, validateTicketExtractionURL(raw))
	}
	_, err := ticketResolvePublic(context.Background(), "127.0.0.1")
	require.Error(t, err)
}

// 公平性用两个模型、一号并发1验证；模型一失败两次不会阻塞模型二直到整个无限任务结束。
type ticketRoundRobinUpstream struct {
	HTTPUpstream
	mu    sync.Mutex
	calls map[string]int
	order []string
}

func (u *ticketRoundRobinUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	var body struct {
		Model string `json:"model"`
	}
	_ = json.NewDecoder(req.Body).Decode(&body)
	u.mu.Lock()
	u.calls[body.Model]++
	n := u.calls[body.Model]
	u.order = append(u.order, body.Model)
	u.mu.Unlock()
	length := 292
	if body.Model == "gpt-6-astra" && n < 3 {
		length = 312
	}
	return ticketResponseForProxy(200, length, ""), nil
}
func TestCodexTicketUnlimitedManualFairnessAndSuccessRounds(t *testing.T) {
	s, repo, _ := setupTicketManualTest(t, 292)
	zero := 0
	one := 1
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{MaxAttempts: &zero, Concurrency: &one}}})
	require.NoError(t, err)
	u := &ticketRoundRobinUpstream{calls: map[string]int{}}
	s.gateway.httpUpstream = u
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	batch, err := s.PrepareManualCollection(ctx, manualRequest(s))
	require.NoError(t, err)
	batch.Execute(func(string, any) bool { return true })
	require.Equal(t, 2, repo.run.Counts["ready"])
	require.Equal(t, 3, u.calls["gpt-6-astra"])
	require.Equal(t, 1, u.calls["gpt-5.6-sol"])
	require.Len(t, u.order, 4)
	require.Contains(t, u.order[:2], "gpt-5.6-sol", "无限模型必须让后面的模型先得到一次机会")
	status, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, 3, status.Items[0].Models[0].Attempts)
	require.Zero(t, status.Items[0].Models[0].MaxAttempts)
}

func TestCodexTicketUnlimitedManualCanCancel(t *testing.T) {
	s, repo, u := setupTicketManualTest(t, 292)
	u.ticket = "gAAAAA" + strings.Repeat("x", 306)
	zero := 0
	models := []string{"gpt-6-astra"}
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{MaxAttempts: &zero, Models: &models}}})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	batch, err := s.PrepareManualCollection(ctx, manualRequest(s))
	require.NoError(t, err)
	batch.Execute(func(string, any) bool { return true })
	require.Equal(t, "cancelled", repo.run.Status)
	require.Equal(t, 1, repo.run.Counts["cancelled"])
	require.Nil(t, s.manualCancel)
	require.Empty(t, s.cache.(*ticketCacheStub).owners)
}

func TestCodexTicketCooldownGatesModelButNeverChangesAccountSwitch(t *testing.T) {
	s, cache, _ := newTicketTestService()
	enableTicketTest(t, s)
	a := ticketAccount()
	a.Schedulable = false
	s.gateway = &OpenAIGatewayService{accountRepo: &ticketAccountStub{accounts: []Account{a}}}
	threshold := 2
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{FailureThreshold: &threshold}}})
	require.NoError(t, err)
	cfg := s.enabledAccountConfig(1)
	key := codexTicketKey(cfg, &a, "gpt-6-astra", "fake-token")
	raw, _ := json.Marshal(codexTicketValue{State: "gAAAAA" + strings.Repeat("x", 286), ExpiresAt: time.Now().Add(time.Hour)})
	encrypted, _ := s.cipher.Encrypt(string(raw))
	cache.values[key] = encrypted
	policy, _ := json.Marshal(map[string]any{"failures": 2, "until": time.Now().Add(time.Minute).UnixMilli()})
	cache.values["collection:"+key] = string(policy)
	require.True(t, s.Blocks(context.Background(), &a, "gpt-6-astra"))
	status, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, "cooldown", status.Items[0].Models[0].State)
	policy, _ = json.Marshal(map[string]any{"failures": 2, "until": time.Now().Add(-time.Second).UnixMilli()})
	cache.values["collection:"+key] = string(policy)
	require.False(t, s.Blocks(context.Background(), &a, "gpt-6-astra"))
	require.False(t, a.Schedulable)
	require.True(t, codexTicketCollectionAllowed(context.Background(), &a))
}
