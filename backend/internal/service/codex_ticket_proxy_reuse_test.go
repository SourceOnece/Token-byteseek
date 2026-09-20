//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 所有上游/取号均为替身；只使用合成凭据，验证真实传入的代理而非仅检查配置。
func setupReuseProxyTest(t *testing.T, mode string, reuse bool) (*CodexTicketService, *ticketCacheStub, *Account) {
	t.Helper()
	s, cache, a := setupTicketProxyTest(t, "rotate", 3)
	rows := []CodexTicketProxyUpdate{
		{ID: "legacy", Name: "A", URL: "http://client-{sid}:synthetic@proxy-a.invalid:8080"},
		{ID: "11111111-1111-4111-8111-111111111111", Name: "B", URL: "http://client-{sid}:synthetic@proxy-b.invalid:8080"},
	}
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{a.ID}, Patch: CodexTicketAccountPatch{ProxyPolicy: &CodexTicketProxyPolicyUpdate{Mode: mode, Proxies: &rows, ReuseSuccessfulIP: &reuse}}})
	require.NoError(t, err)
	return s, cache, a
}

func runReuseAttempt(t *testing.T, s *CodexTicketService, a *Account, model string, p *codexTicketProxy, length int) (CodexTicketAttempt, string) {
	t.Helper()
	u := &ticketSequenceUpstream{responses: []*http.Response{ticketResponseForProxy(200, length, "")}}
	s.gateway.httpUpstream = u
	cfg := s.enabledAccountConfig(a.ID)
	key := codexTicketKey(cfg, a, model, a.GetOpenAIAccessToken())
	var event CodexTicketAttempt
	s.probeAttempt(context.Background(), cfg, a, model, a.GetOpenAIAccessToken(), key, p, 1, func(e CodexTicketAttempt) { event = e })
	require.Len(t, u.proxies, 1)
	return event, u.proxies[0]
}

func TestCodexTicketReuseRotateSuccessFailureAndIsolation(t *testing.T) {
	s, cache, a := setupReuseProxyTest(t, "rotate", true)
	cfg := s.enabledAccountConfig(a.ID)
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	p := cfg.proxies()[1]
	first, address := runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	require.Equal(t, "ready", first.Status)
	require.Equal(t, "new", first.Diagnostic.ProxyUsage)
	cached, _ := cache.Get(context.Background(), "proxy-reuse:"+key)
	require.NotEmpty(t, cached)
	require.NotContains(t, cached, "synthetic")
	require.NotContains(t, cached, "proxy-b.invalid")

	p = cfg.proxies()[0] // 即便入口游标指向A，也优先复用真正成功的B及原SID。
	second, reused := runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	require.Equal(t, address, reused)
	require.Equal(t, "reused", second.Diagnostic.ProxyUsage)
	require.Equal(t, cfg.proxies()[1].ID, p.ID)
	p = cfg.proxies()[0]
	failed, failedAddress := runReuseAttempt(t, s, a, "gpt-6-astra", &p, 356)
	require.Equal(t, address, failedAddress)
	require.Equal(t, "missing", failed.Status)
	require.Equal(t, "reused", failed.Diagnostic.ProxyUsage)
	cleared, _ := cache.Get(context.Background(), "proxy-reuse:"+key)
	require.Empty(t, cleared)
	next, ok := selectCodexTicketProxy(cfg, p.ID, true)
	require.True(t, ok)
	require.Equal(t, cfg.proxies()[0].ID, next.ID)
	third, replacement := runReuseAttempt(t, s, a, "gpt-6-astra", &next, 292)
	require.Equal(t, "new", third.Diagnostic.ProxyUsage)
	require.NotEqual(t, address, replacement)

	// 同代理同凭据，模型不同也不借用Astra的成功会话。
	p = cfg.proxies()[0]
	other, otherAddress := runReuseAttempt(t, s, a, "gpt-5.6-sol", &p, 292)
	require.Equal(t, "new", other.Diagnostic.ProxyUsage)
	require.NotEqual(t, replacement, otherAddress)
	wrong := codexTicketKey(cfg, a, "gpt-5.6-sol", a.GetOpenAIAccessToken())
	_ = cache.Set(context.Background(), "proxy-reuse:"+wrong, cached, time.Hour)
	_, record, err := s.resolveReusableTicketProxy(context.Background(), cfg, wrong, cfg.proxies()[0])
	require.NoError(t, err)
	require.False(t, record.reused, "缓存串键也不能借另一模型的会话")
}

func TestCodexTicketReuseDisabledAndFixedDoNotReadReuseCache(t *testing.T) {
	for _, tc := range []struct {
		mode    string
		enabled bool
	}{{"rotate", false}, {"dynamic", false}, {"fixed", true}} {
		s, cache, a := setupReuseProxyTest(t, tc.mode, tc.enabled)
		cfg := s.enabledAccountConfig(a.ID)
		before := cache.gets
		first, attempt, err := s.resolveReusableTicketProxy(context.Background(), cfg, "not-read", cfg.proxies()[0])
		require.NoError(t, err)
		second, _, err := s.resolveReusableTicketProxy(context.Background(), cfg, "not-read", cfg.proxies()[0])
		require.NoError(t, err)
		require.Empty(t, attempt.key)
		require.Equal(t, before, cache.gets)
		a1, _ := s.cipher.Decrypt(first.Cipher)
		a2, _ := s.cipher.Decrypt(second.Cipher)
		require.NotEqual(t, a1, a2, "关闭时继续沿原每次展开SID逻辑")
	}
}

func TestCodexTicketReuseDynamicProviderOnlyFetchesAfterMiss(t *testing.T) {
	s, _, a := setupReuseProxyTest(t, "dynamic", true)
	endpoint, yes := "https://provider.example/v1/gen?key=synthetic", true
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{a.ID}, Patch: CodexTicketAccountPatch{ProxyPolicy: &CodexTicketProxyPolicyUpdate{Mode: "dynamic", DynamicSource: "api", ExtractionURL: &endpoint, ReuseSuccessfulIP: &yes}}})
	require.NoError(t, err)
	requests := 0
	s.proxyProviderClient = &http.Client{Transport: ticketProviderTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		require.Empty(t, r.Header.Get("Authorization"))
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"proxies":["8.8.8.8:8080:user:synthetic-%d"]}`, requests)))}, nil
	})}
	cfg := s.enabledAccountConfig(a.ID)
	p := cfg.proxies()[0]
	_, first := runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	_, second := runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	require.Equal(t, first, second)
	require.Equal(t, 1, requests)
	_, bad := runReuseAttempt(t, s, a, "gpt-6-astra", &p, 356)
	require.Equal(t, first, bad)
	last, fresh := runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	require.Equal(t, 2, requests)
	require.NotEqual(t, first, fresh)
	require.Equal(t, "new", last.Diagnostic.ProxyUsage)
}

func TestCodexTicketReuseManagedProxyReloadAndDisabled(t *testing.T) {
	s, cache, a := setupReuseProxyTest(t, "rotate", true)
	repo := &ticketManagedProxyRepo{proxy: &Proxy{ID: 9, Protocol: "http", Host: "managed.invalid", Port: 8080, Username: "client-{sid}", Password: "synthetic", Status: StatusActive}}
	s.proxyRepo = repo
	yes := true
	fixed := ""
	rows := []CodexTicketProxyUpdate{{Name: "Managed", ManagedID: 9}}
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{a.ID}, Patch: CodexTicketAccountPatch{ProxyPolicy: &CodexTicketProxyPolicyUpdate{Mode: "rotate", Proxies: &rows, FixedProxyID: &fixed, ReuseSuccessfulIP: &yes}}})
	require.NoError(t, err)
	cfg := s.enabledAccountConfig(a.ID)
	p := cfg.proxies()[0]
	_, first := runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	_, second := runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	require.Equal(t, first, second)
	repo.proxy.Host = "changed.invalid"
	changed, third := runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	require.Equal(t, "new", changed.Diagnostic.ProxyUsage)
	require.Contains(t, third, "changed.invalid")
	repo.proxy.Status = "inactive"
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	_, _, err = s.resolveReusableTicketProxy(context.Background(), cfg, key, cfg.proxies()[0])
	require.Error(t, err)
	cache.fail = true
	_, _, err = s.resolveReusableTicketProxy(context.Background(), cfg, key, cfg.proxies()[0])
	require.ErrorIs(t, err, errTicketProxyReuseCache)
}

func TestCodexTicketReuseNonRetryableRefusalAndStorageKeepRecord(t *testing.T) {
	s, cache, a := setupReuseProxyTest(t, "rotate", true)
	cfg := s.enabledAccountConfig(a.ID)
	p := cfg.proxies()[0]
	_, _ = runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	before, _ := cache.Get(context.Background(), "proxy-reuse:"+key)
	for _, status := range []int{401, 403, 429} {
		u := &ticketSequenceUpstream{responses: []*http.Response{ticketResponseForProxy(status, 0, "")}}
		s.gateway.httpUpstream = u
		var event CodexTicketAttempt
		ready, retry := s.probeAttempt(context.Background(), cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken(), key, &p, 1, func(e CodexTicketAttempt) { event = e })
		require.False(t, ready)
		require.False(t, retry)
		require.Len(t, u.proxies, 1)
		require.Equal(t, "reused", event.Diagnostic.ProxyUsage)
		after, _ := cache.Get(context.Background(), "proxy-reuse:"+key)
		require.Equal(t, before, after)
		cache.values["status:"+key] = "" // 仅替身清退避，以逐项检查每个状态码。
	}
	s.finishTicketProxyReuse(context.Background(), cfg, ticketProxyReuseAttempt{key: key, reused: true}, p, false, false, "storage", nil)
	after, _ := cache.Get(context.Background(), "proxy-reuse:"+key)
	require.Equal(t, before, after)
}

func TestCodexTicketReuseGatewayInheritanceAndSanitizedDiagnostic(t *testing.T) {
	s, _, a := setupTicketProxyTest(t, "rotate", 1)
	yes, no := true, false
	rows := []CodexTicketProxyUpdate{{ID: "legacy", Name: "A"}}
	view, err := s.Update(context.Background(), CodexTicketSettingsUpdate{ProxyPolicy: &CodexTicketProxyPolicyUpdate{Mode: "rotate", Proxies: &rows, ReuseSuccessfulIP: &yes}})
	require.NoError(t, err)
	require.True(t, view.ProxyPolicy.ReuseSuccessfulIP)
	require.True(t, ticketProxyReuseEnabled(s.enabledAccountConfig(a.ID)))
	_, err = s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{a.ID}, Patch: CodexTicketAccountPatch{ProxyPolicy: &CodexTicketProxyPolicyUpdate{Mode: "rotate", Proxies: &rows, ReuseSuccessfulIP: &no}}})
	require.NoError(t, err)
	require.False(t, ticketProxyReuseEnabled(s.enabledAccountConfig(a.ID)))
	_, err = s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{a.ID}, Patch: CodexTicketAccountPatch{ProxyPolicy: &CodexTicketProxyPolicyUpdate{Mode: "inherit"}}})
	require.NoError(t, err)
	require.True(t, ticketProxyReuseEnabled(s.enabledAccountConfig(a.ID)))
	for _, usage := range []string{"new", "reused", "secret-proxy.invalid"} {
		d := safeCodexTicketDiagnostic(&CodexTicketDiagnostic{ProxyUsage: usage})
		raw, _ := json.Marshal(d)
		if usage == "secret-proxy.invalid" {
			require.NotContains(t, string(raw), usage)
		} else {
			require.Equal(t, usage, d.ProxyUsage)
		}
	}
}

// 明确传输失败也丢弃复用出口；取消和存储失败不算代理不合格。
type ticketReuseNetworkError struct{ HTTPUpstream }

func (*ticketReuseNetworkError) Do(*http.Request, string, int64, int) (*http.Response, error) {
	return nil, errors.New("synthetic transport")
}
func TestCodexTicketReuseNetworkFailureDropsSuccessfulRoute(t *testing.T) {
	s, cache, a := setupReuseProxyTest(t, "dynamic", true)
	cfg := s.enabledAccountConfig(a.ID)
	p := cfg.proxies()[0]
	runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	s.gateway.httpUpstream = &ticketReuseNetworkError{}
	ready, retry := s.probeAttempt(context.Background(), cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken(), key, &p, 1, func(CodexTicketAttempt) {})
	require.False(t, ready)
	require.True(t, retry)
	value, _ := cache.Get(context.Background(), "proxy-reuse:"+key)
	require.Empty(t, value)
}

func TestCodexTicketReuseVerifiedFlowPublishesOnlyAfterBothStages(t *testing.T) {
	s, repo, upstream := setupVerifiedTicketTest(t, true)
	rows := []CodexTicketProxyUpdate{{ID: "legacy", Name: "Harvest", URL: "http://client-{sid}:synthetic@harvest.invalid:8080"}}
	yes := true
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{ProxyPolicy: &CodexTicketProxyPolicyUpdate{Mode: "dynamic", Proxies: &rows, ReuseSuccessfulIP: &yes}}})
	require.NoError(t, err)
	a, _ := repo.GetByID(context.Background(), 1)
	cfg := s.enabledAccountConfig(1)
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	var candidate string
	for n, mismatch := range []bool{false, false, true} {
		secondModel := "gpt-6-astra"
		if mismatch {
			secondModel = "gpt-5.6-luna"
		}
		upstream.calls = nil
		upstream.responses = []*http.Response{verifiedResponse(200, 292, "gpt-6-astra"), verifiedResponse(200, 0, secondModel)}
		p := cfg.proxies()[0]
		var event CodexTicketAttempt
		ready, _ := s.probeAttempt(context.Background(), cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken(), key, &p, 1, func(e CodexTicketAttempt) { event = e })
		require.Equal(t, !mismatch, ready)
		require.Len(t, upstream.calls, 2)
		require.Equal(t, a.Proxy.URL(), upstream.calls[1].proxy)
		require.Len(t, upstream.calls[1].state, 292)
		if n == 0 {
			candidate = upstream.calls[0].proxy
		} else {
			require.Equal(t, candidate, upstream.calls[0].proxy)
		}
		require.Len(t, event.Diagnostic.Stages, 2)
		for _, stage := range event.Diagnostic.Stages {
			require.Equal(t, 292, stage.TargetLength)
			require.Equal(t, 312, stage.DegradedSignalLength)
		}
		stored, _ := s.cache.Get(context.Background(), "proxy-reuse:"+key)
		if mismatch {
			require.Empty(t, stored)
		} else {
			require.NotEmpty(t, stored)
		}
	}
}

func TestCodexTicketReuseAcrossInstancesAndCredentialChanges(t *testing.T) {
	s, cache, a := setupReuseProxyTest(t, "dynamic", true)
	cfg := s.enabledAccountConfig(a.ID)
	p := cfg.proxies()[0]
	_, first := runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	// 新服务实例共用持久缓存；不依赖内存中的上一次选择。
	other := &CodexTicketService{cache: cache, cipher: s.cipher, gateway: s.gateway, proxyRepo: s.proxyRepo}
	other.config.Store(s.config.Load())
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	reused, attempt, err := other.resolveReusableTicketProxy(context.Background(), cfg, key, cfg.proxies()[0])
	require.NoError(t, err)
	require.True(t, attempt.reused)
	url, _ := s.cipher.Decrypt(reused.Cipher)
	require.Equal(t, first, url)
	for _, entry := range []struct {
		account      Account
		model, token string
	}{
		{account: *a, model: "gpt-6-astra", token: "changed-synthetic"},
		{account: Account{ID: 2}, model: "gpt-6-astra", token: a.GetOpenAIAccessToken()},
	} {
		newKey := codexTicketKey(cfg, &entry.account, entry.model, entry.token)
		_, newAttempt, err := other.resolveReusableTicketProxy(context.Background(), cfg, newKey, cfg.proxies()[0])
		require.NoError(t, err)
		require.False(t, newAttempt.reused)
	}
	changed := *cfg
	changed.Generation += ":new"
	_, newAttempt, err := other.resolveReusableTicketProxy(context.Background(), &changed, codexTicketKey(&changed, a, "gpt-6-astra", a.GetOpenAIAccessToken()), cfg.proxies()[0])
	require.NoError(t, err)
	require.False(t, newAttempt.reused)
}

func TestCodexTicketReuseExpiredAndCorruptRecordsAreNotReused(t *testing.T) {
	s, cache, a := setupReuseProxyTest(t, "dynamic", true)
	cfg := s.enabledAccountConfig(a.ID)
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	p := cfg.proxies()[0]
	_, _ = runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	encoded, _ := cache.Get(context.Background(), "proxy-reuse:"+key)
	raw, _ := s.cipher.Decrypt(encoded)
	var record ticketProxyReuseRecord
	require.NoError(t, json.Unmarshal([]byte(raw), &record))
	for _, corrupt := range []bool{false, true} {
		value := map[string]any{"scope": record.Scope, "source": record.Source, "proxy": map[string]any{"id": record.Proxy.ID, "name": record.Proxy.Name, "cipher": record.Proxy.Cipher}, "expires_at": time.Now().Add(-time.Minute)}
		if corrupt {
			value["expires_at"] = time.Now().Add(time.Hour)
			value["proxy"].(map[string]any)["managed_proxy_id"] = "invalid-type"
		}
		body, _ := json.Marshal(value)
		broken, _ := s.cipher.Encrypt(string(body))
		require.NoError(t, cache.Set(context.Background(), "proxy-reuse:"+key, broken, time.Hour))
		_, attempt, err := s.resolveReusableTicketProxy(context.Background(), cfg, key, cfg.proxies()[0])
		require.NoError(t, err)
		require.False(t, attempt.reused)
	}
}

func TestCodexTicketReuseStaleSuccessDoesNotOverrideNewerFailure(t *testing.T) {
	s, cache, a := setupReuseProxyTest(t, "rotate", true)
	cfg := s.enabledAccountConfig(a.ID)
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	p := cfg.proxies()[0]
	_, _ = runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	encoded, _ := cache.Get(context.Background(), "proxy-reuse:"+key)
	raw, _ := s.cipher.Decrypt(encoded)
	var saved ticketProxyReuseRecord
	require.NoError(t, json.Unmarshal([]byte(raw), &saved))
	observation := codexTicketObservation{State: "missing", Reason: "invalid_ticket", CheckedAt: saved.ExpiresAt.Add(-ticketProxyReuseTTL).Add(time.Millisecond), Diagnostic: &CodexTicketDiagnostic{ProxyID: p.ID}}
	next, ok := selectCodexTicketProxy(cfg, p.ID, true)
	require.True(t, ok)
	actual, attempt, err := s.resolveReusableTicketProxy(context.Background(), cfg, key, next, observation)
	require.NoError(t, err)
	require.False(t, attempt.reused)
	require.Equal(t, next.ID, actual.ID)
	// 一条更早的失败不能把之后成功的同ID代理否定掉。
	require.NoError(t, cache.Set(context.Background(), "proxy-reuse:"+key, encoded, time.Hour))
	observation.CheckedAt = saved.ExpiresAt.Add(-ticketProxyReuseTTL).Add(-time.Second)
	actual, attempt, err = s.resolveReusableTicketProxy(context.Background(), cfg, key, next, observation)
	require.NoError(t, err)
	require.True(t, attempt.reused)
	require.Equal(t, p.ID, actual.ID)
}

func TestCodexTicketReuseManualAndAutomaticShareSuccessfulRoute(t *testing.T) {
	s, history, _ := setupTicketManualTest(t, 292)
	s.ipProber = &ticketIPStub{ip: "203.0.113.18"}
	yes := true
	models := []string{"gpt-6-astra"}
	rows := []CodexTicketProxyUpdate{{ID: "legacy", Name: "A", URL: "http://client-{sid}:synthetic@manual.invalid:8080"}}
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{Models: &models}, ProxyPolicy: &CodexTicketProxyPolicyUpdate{Mode: "dynamic", Proxies: &rows, ReuseSuccessfulIP: &yes}}})
	require.NoError(t, err)
	a, _ := history.GetByID(context.Background(), 1)
	cfg := s.enabledAccountConfig(1)
	p := cfg.proxies()[0]
	_, first := runReuseAttempt(t, s, a, "gpt-6-astra", &p, 292)
	upstream := &ticketSequenceUpstream{responses: []*http.Response{ticketResponseForProxy(200, 292, "")}}
	s.gateway.httpUpstream = upstream
	m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
	require.NoError(t, err)
	m.Execute(func(string, any) bool { return true })
	m.Close()
	require.Len(t, upstream.proxies, 1)
	require.Equal(t, first, upstream.proxies[0])
	for _, event := range history.events {
		require.Equal(t, "reused", event.Diagnostic.ProxyUsage)
	}
	// 模拟到期续采，只删合成票据缓存，不删成功出口记忆。
	cache := s.cache.(*ticketCacheStub)
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	cache.values[key] = ""
	upstream.responses = []*http.Response{ticketResponseForProxy(200, 292, "")}
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	require.Len(t, upstream.proxies, 2)
	require.Equal(t, first, upstream.proxies[1])
}
