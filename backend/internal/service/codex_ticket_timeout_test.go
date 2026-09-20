//go:build unit

package service

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

// 自定义时限同时进入单次请求、自动轮次和租约；升级读取不换原票据键。
func TestCodexTicketTimeoutRulesAndLegacyKeys(t *testing.T) {
	s, _, _ := setupTicketManualTest(t, 292)
	a := ticketAccount()
	before := codexTicketKey(s.enabledAccountConfig(1), &a, "gpt-6-astra", a.GetOpenAIAccessToken())
	cfg := s.config.Load()
	normalizeTicketConfiguration(cfg)
	require.Equal(t, before, codexTicketKey(s.enabledAccountConfig(1), &a, "gpt-6-astra", a.GetOpenAIAccessToken()))
	view, err := s.AccountSettings(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, 25, view.Rules.AttemptTimeoutSeconds)
	require.Equal(t, time.Minute, s.config.Load().roundTimeout(), "30秒派发窗口之外仍给单次25秒和5秒余量")
	require.Equal(t, 30*time.Second, s.enabledAccountConfig(1).roundTimeout())
	rules := view.Rules
	for _, n := range []int{-1, 0, 4, 301} {
		_, err := applyTicketRulesPatch(rules, &CodexTicketRulesPatch{AttemptTimeoutSeconds: &n})
		require.Error(t, err)
	}
	for _, n := range []int{5, 25, 120, 300} {
		_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{AttemptTimeoutSeconds: &n}}})
		require.NoError(t, err)
		ac := s.enabledAccountConfig(1)
		require.Equal(t, time.Duration(n)*time.Second, ac.attemptTimeout())
		require.Greater(t, ac.collectionLeaseTTL(), ac.attemptTimeout()+15*time.Second)
		require.Greater(t, s.config.Load().roundTimeout(), ac.attemptTimeout())
	}
	gen := s.config.Load().Generation
	n := 90
	_, err = s.UpdateImportDefaults(context.Background(), CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{AttemptTimeoutSeconds: &n}}, nil)
	require.NoError(t, err)
	require.Equal(t, gen, s.config.Load().Generation)
	require.Equal(t, 300*time.Second, s.enabledAccountConfig(1).attemptTimeout(), "改模板不改现有账号")
}

type ticketBudgetUpstream struct {
	HTTPUpstream
	t           *testing.T
	deadlines   []time.Time
	blockVerify bool
}

func (u *ticketBudgetUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	deadline, ok := req.Context().Deadline()
	require.True(u.t, ok)
	u.deadlines = append(u.deadlines, deadline)
	return verifiedResponse(200, 292, "gpt-6-astra"), nil
}
func (u *ticketBudgetUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	deadline, ok := req.Context().Deadline()
	require.True(u.t, ok)
	u.deadlines = append(u.deadlines, deadline)
	if u.blockVerify {
		<-req.Context().Done()
		return nil, req.Context().Err()
	}
	return verifiedResponse(200, 0, "gpt-6-astra"), nil
}

func TestCodexTicketTimeoutBothRoutesShareConfiguredBudget(t *testing.T) {
	s, repo, _ := setupVerifiedTicketTest(t, true)
	n := 120
	err := configureTicketTestAccount(t, s, CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{AttemptTimeoutSeconds: &n}})
	require.NoError(t, err)
	u := &ticketBudgetUpstream{t: t}
	s.gateway.httpUpstream = u
	cfg, a := s.enabledAccountConfig(1), &repo.accounts[0]
	proxy, ok := selectCodexTicketProxy(cfg, "", false)
	require.True(t, ok)
	var event CodexTicketAttempt
	ready, _ := s.probeAttempt(context.Background(), cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken(), codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken()), &proxy, 1, func(e CodexTicketAttempt) { event = e })
	require.True(t, ready)
	require.Len(t, u.deadlines, 2)
	require.Equal(t, u.deadlines[0], u.deadlines[1], "两阶段不是各120秒")
	require.Greater(t, time.Until(u.deadlines[1]), 110*time.Second)
	require.Equal(t, 120, event.Diagnostic.TimeoutSeconds)
	require.Equal(t, "ready", event.Status)
}

func TestCodexTicketTimeoutVerifyFailureDoesNotBecomeCancellation(t *testing.T) {
	s, repo, _ := setupVerifiedTicketTest(t, true)
	n := 5
	require.NoError(t, configureTicketTestAccount(t, s, CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{AttemptTimeoutSeconds: &n}}))
	u := &ticketBudgetUpstream{t: t, blockVerify: true}
	s.gateway.httpUpstream = u
	cfg, a := s.enabledAccountConfig(1), &repo.accounts[0]
	before := a.Schedulable
	proxy, _ := selectCodexTicketProxy(cfg, "", false)
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	var event CodexTicketAttempt
	ready, retry := s.probeAttempt(context.Background(), cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken(), key, &proxy, 1, func(e CodexTicketAttempt) { event = e })
	require.False(t, ready)
	require.True(t, retry)
	require.Equal(t, "failed", event.Status)
	require.Equal(t, "timeout", event.Reason)
	require.Equal(t, "verify", event.Diagnostic.Phase)
	require.Equal(t, "timeout", event.Diagnostic.Stages[1].Reason)
	require.Equal(t, before, repo.accounts[0].Schedulable)
	ticket, err := s.cache.Get(context.Background(), key)
	require.NoError(t, err)
	require.Empty(t, ticket)
}

func TestCodexTicketTimeoutCauseAndDiagnosticSafety(t *testing.T) {
	for _, code := range []string{"timeout", "round_timeout", "lease_lost", "storage", "account_changed", "config_disabled", "config_unavailable", "service_stopped", "client_disconnected"} {
		ctx, stop := context.WithCancelCause(context.Background())
		stop(ticketStopCause(code))
		require.Equal(t, code, ticketContextReason(ctx))
		got := safeTicketLatest(CodexTicketLatest{Source: "manual", State: "cancelled", Reason: code, CheckedAt: time.Now()})
		require.Equal(t, code, got.Reason)
	}
	require.Equal(t, "dns", ticketNetworkKind(&net.DNSError{Err: "secret", Name: "private.invalid"}))
	require.Equal(t, "tls", ticketNetworkKind(x509.UnknownAuthorityError{}))
	require.Equal(t, "connection", ticketNetworkKind(errors.New("http://user:secret@proxy.invalid")))
	d := safeCodexTicketDiagnostic(&CodexTicketDiagnostic{Phase: "secret", NetworkKind: "password", TimeoutSeconds: -1, ElapsedMS: -1, ProviderStatus: 900})
	raw, _ := json.Marshal(d)
	require.NotContains(t, string(raw), "password")
	require.NotContains(t, string(raw), "secret")
	require.Zero(t, d.TimeoutSeconds)
}

func TestCodexTicketTimeoutCurrentPauseReasonsDoNotDependOnScheduling(t *testing.T) {
	a := ticketAccount()
	a.Schedulable = false
	reason, _ := ticketCollectionPause(&a)
	require.Empty(t, reason)
	require.True(t, codexTicketCollectionAllowed(context.Background(), &a))
	at := time.Now().Add(time.Hour)
	a.RateLimitResetAt = &at
	reason, until := ticketCollectionPause(&a)
	require.Equal(t, "account_rate_limited", reason)
	require.Equal(t, &at, until)
	require.False(t, codexTicketCollectionAllowed(context.Background(), &a))
	a.RateLimitResetAt = nil
	a.Status = "inactive"
	reason, _ = ticketCollectionPause(&a)
	require.Equal(t, "account_inactive", reason)
}

func TestCodexTicketTimeoutLongCollectionStaysVisible(t *testing.T) {
	s, _, _ := newTicketTestService()
	raw, _ := json.Marshal(codexTicketObservation{State: "collecting", CheckedAt: time.Now().Add(-70 * time.Second), Diagnostic: &CodexTicketDiagnostic{TimeoutSeconds: 120}})
	got := s.ticketModelStatusForAccount("gpt-6-astra", "", string(raw), time.Now(), &codexTicketConfig{AttemptTimeoutSeconds: 120}, nil)
	require.Equal(t, "collecting", got.State)
	require.Equal(t, "pending", s.ticketModelStatusForAccount("gpt-6-astra", "", string(raw), time.Now(), &codexTicketConfig{}, nil).State)
	require.NotContains(t, strings.ToLower(string(raw)), "token")
}
