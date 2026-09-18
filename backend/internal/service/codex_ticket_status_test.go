//go:build unit

package service

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
	"time"
)

// 状态查询只读真实票据，不调用 token provider/上游，也不把质量状态当票据状态。
func TestCodexTicketStatusIsolationAndReadOnly(t *testing.T) {
	s, cache, _ := newTicketTestService()
	enableTicketTest(t, s)
	a := ticketAccount()
	other := ticketAccount()
	other.ID = 2
	api := ticketAccount()
	api.ID = 3
	api.Type = AccountTypeAPIKey
	repo := &ticketAccountStub{accounts: []Account{a, other, api}}
	s.gateway = &OpenAIGatewayService{accountRepo: repo}
	ticket := seedTicket(t, s, &a, "gpt-6-astra", "fake-token")
	key := codexTicketKey(s.config.Load(), &a, "gpt-5.6-sol", "fake-token")
	s.recordObservation(context.Background(), key, "missing", "invalid_ticket", nil)
	before := len(cache.values)
	result, err := s.Status(context.Background(), []int64{1, 2, 3, 1, 999})
	require.NoError(t, err)
	require.Len(t, result.Items, 3)
	require.Equal(t, "ready", result.Items[0].Models[0].State)
	require.NotNil(t, result.Items[0].Models[0].ExpiresAt)
	require.Equal(t, "missing", result.Items[0].Models[1].State)
	require.Equal(t, "pending", result.Items[1].Models[0].State)
	require.False(t, result.Items[2].Eligible)
	require.Empty(t, result.Items[2].Models)
	require.Equal(t, before, len(cache.values), "查询不写任何缓存")
	require.Empty(t, cache.claims, "查询不发起采集")
	raw, _ := json.Marshal(result)
	for _, secret := range []string{ticket, "fake-token", "workspace-a", "private-password"} {
		require.NotContains(t, string(raw), secret)
	}
	require.True(t, repo.accounts[0].Schedulable)
	require.Empty(t, repo.accounts[0].Extra)
	// 凭据切换后不继承旧票据和状态。
	repo.accounts[0].Credentials = map[string]any{"access_token": "new-token", "chatgpt_account_id": "workspace-a"}
	result, err = s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, "pending", result.Items[0].Models[0].State)
	repo.accounts[0] = a
	enableTicketTest(t, s)
	result, err = s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, "pending", result.Items[0].Models[0].State, "配置换代后不读旧状态")
}

func TestCodexTicketStatusDisabledPausedAndCacheFailure(t *testing.T) {
	s, cache, _ := newTicketTestService()
	a := ticketAccount()
	a.Schedulable = false
	s.gateway = &OpenAIGatewayService{accountRepo: &ticketAccountStub{accounts: []Account{a}}}
	result, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.False(t, result.Enabled)
	require.Equal(t, "disabled", result.Items[0].Models[0].State)
	require.Zero(t, cache.gets)
	enableTicketTest(t, s)
	result, err = s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.True(t, result.Items[0].CollectionPaused)
	require.Equal(t, "paused", result.Items[0].Models[0].State)
	cache.fail = true
	result, err = s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, "unavailable", result.Items[0].Models[0].State, "缓存故障不是采集失败")
	for _, ids := range [][]int64{nil, {0}, {-1}, make([]int64, 101)} {
		_, err = s.Status(context.Background(), ids)
		require.Error(t, err)
	}
}

func TestCodexTicketModelStatusExpiryAndSanitizedReasons(t *testing.T) {
	s, _, _ := newTicketTestService()
	now := time.Now()
	past := now.Add(-time.Second)
	for _, test := range []struct {
		record   codexTicketObservation
		expected string
	}{
		{codexTicketObservation{State: "ready", CheckedAt: past, ExpiresAt: &past}, "expired"},
		{codexTicketObservation{State: "collecting", CheckedAt: past}, "collecting"},
		{codexTicketObservation{State: "collecting", CheckedAt: now.Add(-time.Minute * 2)}, "pending"},
		{codexTicketObservation{State: "failed", Reason: "https://secret:password@host", CheckedAt: past}, "failed"},
	} {
		raw, _ := json.Marshal(test.record)
		got := s.ticketModelStatus("gpt-6-astra", "", string(raw), now)
		require.Equal(t, test.expected, got.State)
		require.Empty(t, got.Reason)
	}
	valid, _ := json.Marshal(codexTicketValue{State: "gAAAAA" + strings.Repeat("x", 286), ExpiresAt: now.Add(time.Hour)})
	encrypted, _ := s.cipher.Encrypt(string(valid))
	failed, _ := json.Marshal(codexTicketObservation{State: "failed", Reason: "network", CheckedAt: now})
	got := s.ticketModelStatus("gpt-6-astra", encrypted, string(failed), now)
	require.Equal(t, "ready", got.State, "刷新失败时旧有效票据仍可用")
	require.Equal(t, "network", got.Reason)
	require.Equal(t, "unavailable", s.ticketModelStatus("gpt-6-astra", "broken", string(failed), now).State)
}

func TestCodexTicketProbeRecordsSafeStatus(t *testing.T) {
	for _, test := range []struct {
		code                  int
		ticket, state, reason string
	}{
		{200, "gAAAAA" + strings.Repeat("x", 286), "ready", ""},
		{200, "not-292", "missing", "invalid_ticket"},
		{503, "raw-upstream-secret", "failed", "upstream"},
	} {
		s, cache, _ := newTicketTestService()
		enableTicketTest(t, s)
		a := ticketAccount()
		up := &ticketUpstreamStub{code: test.code, ticket: test.ticket}
		s.gateway = &OpenAIGatewayService{accountRepo: &ticketAccountStub{accounts: []Account{a}}, httpUpstream: up}
		key := codexTicketKey(s.config.Load(), &a, "gpt-6-astra", "fake-token")
		up.before = func(_ *http.Request) {
			raw, _ := cache.Get(context.Background(), "status:"+key)
			require.Contains(t, raw, `"collecting"`)
		}
		s.probe(context.Background(), s.config.Load(), &a, "gpt-6-astra", "http://proxy")
		raw, err := cache.Get(context.Background(), "status:"+key)
		require.NoError(t, err)
		var record codexTicketObservation
		require.NoError(t, json.Unmarshal([]byte(raw), &record))
		require.Equal(t, test.state, record.State)
		require.Equal(t, test.reason, record.Reason)
		require.NotContains(t, raw, "raw-upstream-secret")
	}
}
