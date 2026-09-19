//go:build unit

package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 可检查的缓存替身只使用合成凭据，记录并发租约与读写，模拟多实例共享 Redis。
type ticketCacheStub struct {
	mu     sync.Mutex
	values map[string]string
	claims map[string]bool
	owners map[string]string
	gets   int
	fail   bool
	onGet  func()
}

func (c *ticketCacheStub) Get(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gets++
	if c.onGet != nil {
		c.onGet()
	}
	if c.fail {
		return "", errors.New("offline")
	}
	return c.values[key], nil
}
func (c *ticketCacheStub) Set(_ context.Context, key, value string, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return errors.New("offline")
	}
	c.values[key] = value
	return nil
}
func (c *ticketCacheStub) Claim(_ context.Context, key string, _ time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return false, errors.New("offline")
	}
	if c.claims[key] {
		return false, nil
	}
	c.claims[key] = true
	return true, nil
}

type ticketCipherStub struct{}

func (c *ticketCacheStub) SetLatest(_ context.Context, key, value string, finishedUS int64, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return errors.New("offline")
	}
	var previous CodexTicketLatest
	_ = json.Unmarshal([]byte(c.values["latest:"+key]), &previous)
	if previous.FinishedUS > finishedUS {
		return nil
	}
	c.values["latest:"+key] = value
	return nil
}

func (c *ticketCacheStub) AcquireLease(_ context.Context, key, owner string, _ time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return false, errors.New("offline")
	}
	if c.owners == nil {
		c.owners = map[string]string{}
	}
	if c.owners[key] != "" {
		return false, nil
	}
	c.owners[key] = owner
	return true, nil
}
func (c *ticketCacheStub) ReleaseLease(_ context.Context, key, owner string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.owners[key] == owner {
		delete(c.owners, key)
	}
	return nil
}

func (c *ticketCacheStub) RenewLease(_ context.Context, key, owner string, _ time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return false, errors.New("offline")
	}
	return c.owners[key] == owner, nil
}

func (c *ticketCacheStub) LeaseValue(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return "", errors.New("offline")
	}
	return c.owners[key], nil
}

func (c *ticketCacheStub) ReplaceLease(_ context.Context, key, owner, replacement string, _ time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return false, errors.New("offline")
	}
	if c.owners[key] != owner {
		return false, nil
	}
	c.owners[key] = replacement
	return true, nil
}

func (c *ticketCacheStub) GetMany(ctx context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		value, err := c.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		out[key] = value
	}
	return out, nil
}

func (ticketCipherStub) Encrypt(value string) (string, error) {
	return base64.StdEncoding.EncodeToString([]byte(value)), nil
}
func (ticketCipherStub) Decrypt(value string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(value)
	return string(raw), err
}

type ticketSettingStub struct {
	*mockSettingRepo
	writeFail bool
}

func (r *ticketSettingStub) GetValue(ctx context.Context, key string) (string, error) {
	v, e := r.mockSettingRepo.GetValue(ctx, key)
	if e == nil && v == "" {
		e = ErrSettingNotFound
	}
	return v, e
}
func (r *ticketSettingStub) Set(ctx context.Context, key, value string) error {
	if r.writeFail {
		return errors.New("db failure")
	}
	return r.mockSettingRepo.Set(ctx, key, value)
}

type ticketAccountStub struct {
	AccountRepository
	mu       sync.Mutex
	accounts []Account
}

func (r *ticketAccountStub) ListByPlatform(context.Context, string) ([]Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Account(nil), r.accounts...), nil
}
func (r *ticketAccountStub) GetByIDs(ctx context.Context, ids []int64) ([]*Account, error) {
	out := []*Account{}
	for _, id := range ids {
		a, err := r.GetByID(ctx, id)
		if err == nil {
			out = append(out, a)
		}
	}
	return out, nil
}
func (r *ticketAccountStub) GetByID(_ context.Context, id int64) (*Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.accounts {
		if a.ID == id {
			copy := a
			return &copy, nil
		}
	}
	return nil, errors.New("not found")
}

type ticketUpstreamStub struct {
	HTTPUpstream
	calls  atomic.Int32
	before func(*http.Request)
	code   int
	ticket string
}

func (u *ticketUpstreamStub) Do(req *http.Request, proxy string, _ int64, _ int) (*http.Response, error) {
	u.calls.Add(1)
	if u.before != nil {
		u.before(req)
	}
	if req.Context().Err() != nil {
		return nil, req.Context().Err()
	}
	return &http.Response{StatusCode: u.code, Header: http.Header{http.CanonicalHeaderKey(openAICodexTurnStateHeader): {u.ticket}}, Body: io.NopCloser(strings.NewReader(""))}, nil
}
func (u *ticketUpstreamStub) DoWithTLS(req *http.Request, proxy string, id int64, c int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxy, id, c)
}
func ticketAccount() Account {
	return Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"access_token": "fake-token", "chatgpt_account_id": "workspace-a"}}
}
func newTicketTestService() (*CodexTicketService, *ticketCacheStub, *ticketSettingStub) {
	cache := &ticketCacheStub{values: map[string]string{}, claims: map[string]bool{}}
	settings := &ticketSettingStub{mockSettingRepo: newMockSettingRepo()}
	s := &CodexTicketService{settings: settings, cache: cache, cipher: ticketCipherStub{}}
	return s, cache, settings
}
func enableTicketTest(t *testing.T, s *CodexTicketService) {
	t.Helper()
	yes := true
	proxy := "socks5h://example:private-password@proxy.example:1080"
	_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{Enabled: &yes, HarvestProxyURL: &proxy})
	require.NoError(t, err)
}
func seedTicket(t *testing.T, s *CodexTicketService, a *Account, model, token string) string {
	t.Helper()
	ticket := "gAAAAA" + strings.Repeat("a", 286)
	raw, _ := json.Marshal(codexTicketValue{State: ticket, ExpiresAt: time.Now().Add(time.Hour)})
	encoded, err := s.cipher.Encrypt(string(raw))
	require.NoError(t, err)
	require.NoError(t, s.cache.Set(context.Background(), codexTicketKey(s.config.Load(), a, model, token), encoded, time.Hour))
	return ticket
}

// 三种出站构建器共用票据注入，不更改会话隔离、鉴权、路由提示或请求正文。
func TestCodexTicketGatewayBuildersPreserveLegacyHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, path := range []string{"http", "passthrough", "ws"} {
		t.Run(path, func(t *testing.T) {
			s, _, _ := newTicketTestService()
			enableTicketTest(t, s)
			a := ticketAccount()
			ticket := seedTicket(t, s, &a, "gpt-6-astra", "fake-token")
			g := &OpenAIGatewayService{}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			c.Request.Header.Set("session_id", "client-session")
			c.Request.Header.Set("conversation_id", "client-conversation")
			body := []byte(`{"model":"gpt-6-astra","service_tier":"priority","input":"synthetic"}`)
			build := func() http.Header {
				if path == "ws" {
					h, _, err := g.buildOpenAIWSHeaders(context.Background(), c, &a, "fake-token", OpenAIWSProtocolDecision{Transport: OpenAIUpstreamTransportResponsesWebsocketV2}, true, "", "", "cache", "gpt-6-astra", "priority")
					require.NoError(t, err)
					return h
				}
				var req *http.Request
				var err error
				if path == "http" {
					req, err = g.buildUpstreamRequest(context.Background(), c, &a, body, "fake-token", false, "cache", true)
				} else {
					req, err = g.buildUpstreamRequestOpenAIPassthrough(context.Background(), c, &a, body, "fake-token")
				}
				require.NoError(t, err)
				wire, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				require.Equal(t, body, wire)
				return req.Header
			}
			legacy := build()
			g.codexTickets.Store(s)
			injected := build()
			require.Equal(t, ticket, injected.Get(openAICodexTurnStateHeader))
			injected.Del(openAICodexTurnStateHeader)
			require.Equal(t, legacy, injected)
			no := false
			_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{Enabled: &no})
			require.NoError(t, err)
			require.Equal(t, legacy, build())
		})
	}
}

func TestCodexTicketSettingsDefaultOffAndGeneration(t *testing.T) {
	s, _, r := newTicketTestService()
	ctx := context.Background()
	v, err := s.View(ctx)
	require.NoError(t, err)
	require.False(t, v.Enabled)
	require.False(t, v.ProxyConfigured)
	yes := true
	_, err = s.Update(ctx, CodexTicketSettingsUpdate{Enabled: &yes})
	require.Error(t, err)
	enableTicketTest(t, s)
	first := s.config.Load().Generation
	v, err = s.View(ctx)
	require.NoError(t, err)
	require.True(t, v.Enabled)
	require.True(t, v.ProxyConfigured)
	raw, _ := json.Marshal(v)
	require.NotContains(t, string(raw), "private-password")
	stored, err := r.GetValue(ctx, codexTicketSettingsKey)
	require.NoError(t, err)
	require.NotContains(t, stored, "private-password")
	no := false
	_, err = s.Update(ctx, CodexTicketSettingsUpdate{Enabled: &no})
	require.NoError(t, err)
	require.Nil(t, s.enabledConfig())
	require.NotEqual(t, first, s.config.Load().Generation)
	_, err = s.Update(ctx, CodexTicketSettingsUpdate{Enabled: &no, ClearProxy: true})
	require.NoError(t, err)
	v, err = s.View(ctx)
	require.NoError(t, err)
	require.False(t, v.ProxyConfigured)
	r.writeFail = true
	_, err = s.Update(ctx, CodexTicketSettingsUpdate{Enabled: &no})
	require.Error(t, err)
	require.Nil(t, s.enabledConfig())
}

func TestCodexTicketProxyValidationRedactsErrors(t *testing.T) {
	for _, raw := range []string{"http://u:secret@proxy/x", "ftp://secret@proxy", "socks5h://proxy:70000", "https://proxy?password=secret", "http://proxy/#secret"} {
		err := validateCodexHarvestProxy(raw)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "secret")
	}
	for _, raw := range []string{"http://proxy:80", "https://u:p@proxy:443", "socks5://proxy", "socks5h://proxy:1080"} {
		require.NoError(t, validateCodexHarvestProxy(raw))
	}
}

func TestCodexTicketApplyOptInIsolationAndExistingState(t *testing.T) {
	s, cache, _ := newTicketTestService()
	a := ticketAccount()
	ctx := context.Background()
	headers := http.Header{"Authorization": {"Bearer fake-token"}}
	s.Apply(ctx, &a, "gpt-6-astra", headers)
	require.Zero(t, cache.gets)
	require.Empty(t, headers.Get(openAICodexTurnStateHeader))
	enableTicketTest(t, s)
	ticket := seedTicket(t, s, &a, "gpt-6-astra", "fake-token")
	s.Apply(ctx, &a, "gpt-6-astra", headers)
	require.Equal(t, ticket, headers.Get(openAICodexTurnStateHeader))
	for _, tc := range []struct {
		name   string
		mutate func(*Account, http.Header)
		model  string
	}{
		{"different account", func(a *Account, _ http.Header) { a.ID = 2 }, "gpt-6-astra"},
		{"new credential", func(_ *Account, h http.Header) { h.Set("Authorization", "Bearer rotated") }, "gpt-6-astra"},
		{"new workspace", func(a *Account, _ http.Header) { a.Credentials = map[string]any{"chatgpt_account_id": "workspace-b"} }, "gpt-6-astra"},
		{"api key", func(a *Account, _ http.Header) { a.Type = AccountTypeAPIKey }, "gpt-6-astra"},
		{"non target", func(*Account, http.Header) {}, "gpt-5.5"},
		{"compact mapped sol without sol ticket", func(*Account, http.Header) {}, "gpt-5.6-sol"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copy := a
			h := http.Header{"Authorization": {"Bearer fake-token"}}
			tc.mutate(&copy, h)
			s.Apply(ctx, &copy, tc.model, h)
			require.Empty(t, h.Get(openAICodexTurnStateHeader))
		})
	}
	for _, key := range []string{"X-Codex-Turn-State", "x-codex-turn-state"} {
		h := http.Header{"Authorization": {"Bearer fake-token"}, key: {"client-state"}}
		require.NoError(t, s.Apply(ctx, &a, "gpt-6-astra", h))
		require.Equal(t, ticket, h.Get(openAICodexTurnStateHeader))
		require.Len(t, h, 2)
	}
	no := false
	_, err := s.Update(ctx, CodexTicketSettingsUpdate{Enabled: &no})
	require.NoError(t, err)
	enableTicketTest(t, s)
	h := http.Header{"Authorization": {"Bearer fake-token"}}
	s.Apply(ctx, &a, "gpt-6-astra", h)
	require.Empty(t, h.Get(openAICodexTurnStateHeader), "新代次不读旧票据")
	require.True(t, a.Schedulable)
}

func TestCodexTicketReadFailureAndDisableDuringRead(t *testing.T) {
	s, cache, _ := newTicketTestService()
	enableTicketTest(t, s)
	a := ticketAccount()
	seedTicket(t, s, &a, "gpt-6-astra", "fake-token")
	cache.fail = true
	h := http.Header{"Authorization": {"Bearer fake-token"}}
	require.ErrorIs(t, s.Apply(context.Background(), &a, "gpt-6-astra", h), ErrCodexTicketUnavailable)
	require.Empty(t, h.Get(openAICodexTurnStateHeader))
	cache.fail = false
	s.cacheRetryAt.Store(0)
	cache.onGet = func() { s.config.Store(&codexTicketConfig{loadedAt: time.Now()}) }
	require.NoError(t, s.Apply(context.Background(), &a, "gpt-6-astra", h))
	require.Empty(t, h.Get(openAICodexTurnStateHeader))
}

func TestCodexTicketHarvestIsBoundedAndSchedulingDisabledAccountsIncluded(t *testing.T) {
	s, cache, _ := newTicketTestService()
	enableTicketTest(t, s)
	active, paused := ticketAccount(), ticketAccount()
	paused.ID = 2
	paused.Schedulable = false
	repo := &ticketAccountStub{accounts: []Account{active, paused}}
	up := &ticketUpstreamStub{code: 200, ticket: "gAAAAA" + strings.Repeat("x", 286)}
	up.before = func(req *http.Request) {
		require.Equal(t, HTTPUpstreamProfileOpenAIHarvest, HTTPUpstreamProfileFromContext(req.Context()))
		require.True(t, HTTPUpstreamRedirectsDisabled(req.Context()))
		require.True(t, req.Close)
		require.Equal(t, "chatgpt.com", req.URL.Host)
	}
	s.gateway = &OpenAIGatewayService{accountRepo: repo, httpUpstream: up}
	s.harvest(context.Background())
	require.Equal(t, int32(4), up.calls.Load())
	require.Len(t, cache.values, 12)
	s.harvest(context.Background())
	require.Equal(t, int32(4), up.calls.Load(), "已有有效票不重复采集")
	require.False(t, repo.accounts[1].Schedulable)
	require.Empty(t, repo.accounts[0].Extra)
	no := false
	_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{Enabled: &no})
	require.NoError(t, err)
	s.harvest(context.Background())
	require.Equal(t, int32(4), up.calls.Load())
}

func TestCodexTicketChangedCredentialsNotPersisted(t *testing.T) {
	s, cache, _ := newTicketTestService()
	enableTicketTest(t, s)
	a := ticketAccount()
	repo := &ticketAccountStub{accounts: []Account{a}}
	up := &ticketUpstreamStub{code: 200, ticket: "gAAAAA" + strings.Repeat("x", 286), before: func(*http.Request) {
		repo.mu.Lock()
		repo.accounts[0].Credentials = map[string]any{"access_token": "rotated"}
		repo.mu.Unlock()
	}}
	s.gateway = &OpenAIGatewayService{accountRepo: repo, httpUpstream: up}
	s.probe(context.Background(), s.config.Load(), &a, "gpt-6-astra")
	for key := range cache.values {
		require.True(t, strings.HasPrefix(key, "status:") || strings.HasPrefix(key, "latest:"), "换凭据后的旧结果只能记录隔离诊断，不得保存可用票据")
	}
}

func TestCodexTicketStopCancelsProbeAndWaits(t *testing.T) {
	s, _, _ := newTicketTestService()
	enableTicketTest(t, s)
	a := ticketAccount()
	repo := &ticketAccountStub{accounts: []Account{a}}
	started := make(chan struct{}, 2)
	up := &ticketUpstreamStub{code: 200, before: func(req *http.Request) { started <- struct{}{}; <-req.Context().Done() }}
	s.gateway = &OpenAIGatewayService{accountRepo: repo, httpUpstream: up}
	s.Start()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("probe did not start")
	}
	done := make(chan struct{})
	go func() { s.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stop did not cancel workers")
	}
	s.Start()
	s.Stop()
}
