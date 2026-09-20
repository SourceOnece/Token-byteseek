//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 合成历史仓储验证只写脱敏事件，不借用账号质量结果或调度存储。
type ticketHistoryStub struct {
	*ticketAccountStub
	events     []CodexTicketAttempt
	run        *CodexTicketManualRun
	appendFail bool
}

func (r *ticketHistoryStub) CreateTicketRun(_ context.Context, run *CodexTicketManualRun) error {
	copy := *run
	r.run = &copy
	return nil
}
func (r *ticketHistoryStub) AppendTicketEvent(_ context.Context, _ string, e *CodexTicketAttempt) error {
	if r.appendFail {
		return errors.New("offline")
	}
	e.ID = int64(len(r.events) + 1)
	r.events = append(r.events, *e)
	return nil
}
func (r *ticketHistoryStub) HeartbeatTicketRun(context.Context, string) error { return nil }
func (r *ticketHistoryStub) FinishTicketRun(_ context.Context, _ string, status string, counts map[string]int) error {
	r.run.Status = status
	r.run.Counts = counts
	return nil
}
func (r *ticketHistoryStub) ListTicketRuns(context.Context, int) ([]CodexTicketManualRun, error) {
	return nil, nil
}
func (r *ticketHistoryStub) GetTicketRun(context.Context, string) (*CodexTicketManualRun, error) {
	return r.run, nil
}
func (r *ticketHistoryStub) ListTicketEvents(context.Context, string, string, string, int64, string, int) ([]CodexTicketAttempt, int, error) {
	return r.events, len(r.events), nil
}

type ticketIPStub struct {
	mu    sync.Mutex
	calls int
	ip    string
}

func (p *ticketIPStub) ProbeProxy(context.Context, string) (*ProxyExitInfo, int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	return &ProxyExitInfo{IP: p.ip}, 1, nil
}

func setupTicketManualTest(t *testing.T, length int) (*CodexTicketService, *ticketHistoryStub, *ticketUpstreamStub) {
	t.Helper()
	s, _, _ := newTicketTestService()
	enableTicketTest(t, s)
	if length != 292 {
		_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{TargetLength: &length})
		require.NoError(t, err)
	}
	a := ticketAccount()
	a.Credentials["email"] = "synthetic@example.invalid"
	r := &ticketHistoryStub{ticketAccountStub: &ticketAccountStub{accounts: []Account{a}}}
	u := &ticketUpstreamStub{code: 200, ticket: "gAAAAA" + strings.Repeat("x", length-6)}
	s.gateway = &OpenAIGatewayService{accountRepo: r, httpUpstream: u}
	if length != 292 {
		require.NoError(t, configureTicketTestAccount(t, s, CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{TargetLength: &length}}))
	}
	return s, r, u
}
func manualRequest(s *CodexTicketService) CodexTicketManualRequest {
	return CodexTicketManualRequest{AccountIDs: []int64{1}, Confirmed: true, Revision: s.config.Load().Generation}
}

func TestCodexTicketManualLogsBothModelsAndCustomLength(t *testing.T) {
	s, r, u := setupTicketManualTest(t, 332)
	s.ipProber = &ticketIPStub{ip: "203.0.113.18"}
	m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
	require.NoError(t, err)
	_, err = s.PrepareManualCollection(context.Background(), manualRequest(s))
	require.Error(t, err)
	var kinds []string
	m.Execute(func(kind string, _ any) bool { kinds = append(kinds, kind); return true })
	require.Equal(t, int32(2), u.calls.Load())
	require.Len(t, r.events, 4)
	require.Equal(t, 2, r.run.Counts["ready"])
	require.Equal(t, "completed", r.run.Status)
	require.Equal(t, "start", kinds[0])
	require.Equal(t, "complete", kinds[len(kinds)-1])
	for _, e := range r.events {
		require.Equal(t, 332, e.TargetLength)
		require.Equal(t, "203.0.113.18", e.ReferenceIP)
		require.Equal(t, "reference", e.IPStatus)
		require.Equal(t, "synthetic@example.invalid", e.Email)
		require.Equal(t, 332, e.Diagnostic.HeaderLength)
	}
	raw, _ := json.Marshal(r.events)
	for _, secret := range []string{u.ticket, "fake-token", "private-password", "workspace-a", "proxy.example"} {
		require.NotContains(t, string(raw), secret)
	}
	a := r.accounts[0]
	require.False(t, s.Blocks(context.Background(), &a, "gpt-6-astra"))
	require.True(t, a.Schedulable)
	require.Empty(t, a.Extra)
	h := http.Header{"Authorization": {"Bearer fake-token"}}
	require.NoError(t, s.Apply(context.Background(), &a, "gpt-6-astra", h))
	require.Len(t, h.Get(openAICodexTurnStateHeader), 332)
	status, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, 332, status.Items[0].Models[0].TargetLength)
	require.Equal(t, "ready", status.Items[0].Models[0].State)
	cache := s.cache.(*ticketCacheStub)
	require.Empty(t, cache.owners)
	require.Nil(t, s.manualCancel)
}

// 停调号可自动或手动采集，均不打开真实账号调度。
func TestCodexTicketManualSchedulingDisabledStillCollects(t *testing.T) {
	s, r, u := setupTicketManualTest(t, 292)
	r.accounts[0].Schedulable = false
	a := r.accounts[0]
	s.probe(context.Background(), s.config.Load(), &a, "gpt-6-astra")
	require.Equal(t, int32(1), u.calls.Load(), "自动采集也允许停调账号")
	m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
	require.NoError(t, err)
	m.Execute(func(string, any) bool { return true })
	require.Equal(t, int32(3), u.calls.Load())
	require.Equal(t, 2, r.run.Counts["ready"])
	require.False(t, r.accounts[0].Schedulable)
	require.False(t, r.accounts[0].IsSchedulable())
	require.Empty(t, r.accounts[0].Extra)
	status, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, "ready", status.Items[0].Models[0].State)
	require.Equal(t, "manual", status.Items[0].Models[0].Latest.Source)
}
func TestCodexTicketManualDoesNotIgnoreOtherEligibility(t *testing.T) {
	ctx := context.WithValue(context.Background(), codexTicketManualContextKey{}, true)
	a := ticketAccount()
	a.Schedulable = false
	require.True(t, codexTicketCollectionAllowed(ctx, &a))
	deadline := time.Now().Add(time.Hour)
	for _, mutate := range []func(*Account){
		func(a *Account) { a.Status = "inactive" },
		func(a *Account) { a.RateLimitResetAt = &deadline },
		func(a *Account) { a.Type = AccountTypeAPIKey },
	} {
		copy := a
		mutate(&copy)
		require.False(t, codexTicketCollectionAllowed(ctx, &copy))
	}
	require.True(t, codexTicketCollectionAllowed(context.Background(), &a))
}
func TestCodexTicketManualCancellationAndStorageFailure(t *testing.T) {
	for _, mode := range []string{"disconnect", "storage", "disable"} {
		t.Run(mode, func(t *testing.T) {
			s, r, u := setupTicketManualTest(t, 292)
			m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
			require.NoError(t, err)
			if mode == "storage" {
				r.appendFail = true
			}
			if mode == "disable" {
				no := false
				_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{Enabled: &no})
				require.NoError(t, err)
			}
			m.Execute(func(kind string, _ any) bool { return !(mode == "disconnect" && kind == "start") })
			if mode == "storage" {
				require.Equal(t, "failed", r.run.Status)
			} else {
				require.Equal(t, "cancelled", r.run.Status)
				require.Zero(t, u.calls.Load())
			}
			require.Nil(t, s.manualCancel)
			require.Empty(t, s.cache.(*ticketCacheStub).owners)
		})
	}
}
func TestCodexTicketManualRequiresConfirmationRevisionAndEnabled(t *testing.T) {
	s, _, u := setupTicketManualTest(t, 292)
	req := manualRequest(s)
	req.Confirmed = false
	_, err := s.PrepareManualCollection(context.Background(), req)
	require.Error(t, err)
	req = manualRequest(s)
	req.Revision = "stale"
	_, err = s.PrepareManualCollection(context.Background(), req)
	require.Error(t, err)
	no := false
	_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{Enabled: &no})
	require.NoError(t, err)
	_, err = s.PrepareManualCollection(context.Background(), manualRequest(s))
	require.Error(t, err)
	require.Zero(t, u.calls.Load())
}
func TestCodexTicketManualHonorsBackoffAndKeepsOldTicket(t *testing.T) {
	s, r, u := setupTicketManualTest(t, 292)
	a := r.accounts[0]
	seedTicket(t, s, &a, "gpt-6-astra", "fake-token")
	deadline := time.Now().Add(time.Minute)
	key := codexTicketKey(s.config.Load(), &a, "gpt-6-astra", "fake-token")
	s.recordObservation(context.Background(), key, "failed", "upstream", nil, &CodexTicketDiagnostic{RetryNotBefore: &deadline})
	m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
	require.NoError(t, err)
	m.Execute(func(string, any) bool { return true })
	require.Equal(t, int32(1), u.calls.Load())
	require.Equal(t, 1, r.run.Counts["skipped"])
	require.False(t, s.Blocks(context.Background(), &a, "gpt-6-astra"))
}
func TestCodexTicketManualAttemptsCanExceedTen(t *testing.T) {
	s, r, u := setupTicketManualTest(t, 292)
	count := 12
	err := configureTicketTestAccount(t, s, CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{MaxAttempts: &count}})
	require.NoError(t, err)
	// 单模型顺序测试：第十二次才命中，配置有效时间使用合成未来时间避免启动自动 runner。
	cfg := *s.config.Load()
	cfg.loadedAt = time.Now().Add(time.Minute)
	s.config.Store(&cfg)
	u.ticket = "gAAAAA" + strings.Repeat("x", 306)
	u.before = func(*http.Request) {
		if u.calls.Load() == 12 {
			u.ticket = "gAAAAA" + strings.Repeat("x", 286)
		}
	}
	m := &CodexTicketManualSession{s: s, cfg: &cfg, repo: r}
	events := []CodexTicketAttempt{}
	m.collectModel(context.Background(), 1, "gpt-6-astra", func(e CodexTicketAttempt) { events = append(events, e) })
	require.Len(t, events, 13)
	require.Equal(t, 12, events[11].Attempt)
	require.Equal(t, "ready", events[12].Status)
}
