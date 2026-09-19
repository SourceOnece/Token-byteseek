//go:build unit

package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 用同一仓储覆盖真实手动/自动入口，不以独立setter测试替代采集后的联动。
type ticketSchedulingRepo struct {
	*ticketHistoryStub
	writes int
	fail   bool
}

func (r *ticketSchedulingRepo) ApplyCodexTicketScheduling(_ context.Context, expected *Account, enabled bool) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail {
		return false, errors.New("synthetic database failure")
	}
	for i := range r.accounts {
		a := &r.accounts[i]
		if a.ID == expected.ID && a.UpdatedAt.Equal(expected.UpdatedAt) && a.Schedulable != enabled {
			a.Schedulable = enabled
			a.UpdatedAt = time.Now()
			r.writes++
			return true, nil
		}
	}
	return false, nil
}

func TestCodexTicketSchedulingLengthMatrixManualAndAutomatic(t *testing.T) {
	for _, source := range []string{"manual", "auto"} {
		for _, initial := range []bool{false, true} {
			for _, tc := range []struct {
				name         string
				length, code int
				want         int
			}{
				{"qualified", 332, 200, 1}, {"degraded", 312, 200, 0}, {"other", 356, 200, -1},
				{"http_error_qualified", 332, 503, -1}, {"http_error_signal", 312, 503, -1}, {"no_header", 0, 200, -1},
			} {
				t.Run(source+"/"+tc.name+map[bool]string{false: "/off", true: "/on"}[initial], func(t *testing.T) {
					s, base, u := setupTicketManualTest(t, 332)
					base.accounts[0].Schedulable = initial
					r := &ticketSchedulingRepo{ticketHistoryStub: base}
					s.gateway.accountRepo = r
					models := []string{"gpt-6-astra"}
					signal := 312
					_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{Models: &models, DegradedSignalLength: &signal}}})
					require.NoError(t, err)
					u.code = tc.code
					u.ticket = ""
					if tc.length > 0 {
						u.ticket = "gAAAAA" + strings.Repeat("x", tc.length-6)
					}
					if source == "auto" {
						a, _ := r.GetByID(context.Background(), 1)
						s.probe(context.Background(), s.config.Load(), a, models[0])
					} else {
						batch, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
						require.NoError(t, err)
						batch.Execute(func(string, any) bool { return true })
					}
					a, err := r.GetByID(context.Background(), 1)
					require.NoError(t, err)
					want := initial
					if tc.want >= 0 {
						want = tc.want == 1
					}
					require.Equal(t, want, a.Schedulable)
					if want == initial {
						require.Zero(t, r.writes)
					} else {
						require.Equal(t, 1, r.writes)
					}
				})
			}
		}
	}
}

func TestCodexTicketSchedulingFailureStaleAndStorageGuard(t *testing.T) {
	s, base, _ := setupTicketManualTest(t, 292)
	base.accounts[0].Schedulable = false
	r := &ticketSchedulingRepo{ticketHistoryStub: base}
	s.gateway.accountRepo = r
	a, _ := r.GetByID(context.Background(), 1)
	cfg := s.enabledAccountConfig(1)
	r.fail = true
	require.Equal(t, "failed", s.applyTicketScheduling(context.Background(), cfg, a, true))
	r.fail = false
	base.accounts[0].UpdatedAt = time.Now()
	require.Equal(t, "stale", s.applyTicketScheduling(context.Background(), cfg, a, true))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Equal(t, "stale", s.applyTicketScheduling(ctx, cfg, a, true))
	require.Zero(t, r.writes)

	// 缓存写失败不能把尚未发布票据的账号打开。
	s.cache = &ticketSchedulingWriteFailure{ticketCacheStub: s.cache.(*ticketCacheStub)}
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	require.Zero(t, r.writes)
}

// 只让真正票据发布失败，读缓存/状态/租约仍可用，确保覆盖收到合格头之后的保护。
type ticketSchedulingWriteFailure struct{ *ticketCacheStub }

func (c *ticketSchedulingWriteFailure) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if len(key) == 64 {
		return errors.New("synthetic ticket storage failure")
	}
	return c.ticketCacheStub.Set(ctx, key, value, ttl)
}

func TestCodexTicketBusinessSchedulingLengthAndReceiptGuards(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		length               int
		initial, want        bool
		replace, change, off bool
	}{
		{"qualified", 292, false, true, false, false, false},
		{"degraded", 312, true, false, false, false, false},
		{"other", 356, true, true, false, false, false},
		{"other_paused", 356, false, false, false, false, false},
		{"new_ticket", 312, true, true, true, false, false},
		{"new_account_change", 312, true, true, false, true, false},
		{"watchdog_off", 312, true, true, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, base, _ := setupTicketManualTest(t, 292)
			base.accounts[0].Schedulable = tc.initial
			r := &ticketSchedulingRepo{ticketHistoryStub: base}
			s.gateway.accountRepo = r
			signal := 312
			mode := "observe"
			if tc.off {
				mode = "off"
			}
			_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{WatchdogMode: &mode, Rules: &CodexTicketRulesPatch{DegradedSignalLength: &signal}}})
			require.NoError(t, err)
			a, _ := r.GetByID(context.Background(), 1)
			cfg := s.enabledAccountConfig(1)
			// seedTicket针对全局key，当前用完整有效账号cfg构造本次真实收据。
			key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
			receipt := &codexTicketReceipt{s: s, cfg: cfg, key: key, encoded: "synthetic-encrypted-ticket", model: "gpt-6-astra"}
			s.cache.(*ticketCacheStub).values[key] = receipt.encoded
			receipt.observeHeader(http.Header{http.CanonicalHeaderKey(openAICodexTurnStateHeader): {"gAAAAA" + strings.Repeat("x", tc.length-6)}})
			if tc.replace {
				s.cache.(*ticketCacheStub).values[key] = "new-encrypted-ticket"
			}
			if tc.change {
				base.accounts[0].UpdatedAt = time.Now()
			}
			for _, event := range s.schedulingPending {
				s.applyTicketSchedulingSignal(context.Background(), event)
			}
			a, _ = r.GetByID(context.Background(), 1)
			require.Equal(t, tc.want, a.Schedulable)
		})
	}
}

func TestCodexTicketSchedulingQueueBoundedAndLatestWins(t *testing.T) {
	s, base, _ := setupTicketManualTest(t, 292)
	base.accounts[0].Schedulable = true
	r := &ticketSchedulingRepo{ticketHistoryStub: base}
	s.gateway.accountRepo = r
	cfg := s.enabledAccountConfig(1)
	receipt := &codexTicketReceipt{s: s, cfg: cfg, key: "synthetic", encoded: "same", model: "gpt-6-astra"}
	s.queueTicketScheduling(receipt, true)
	s.queueTicketScheduling(receipt, false)
	require.Len(t, s.schedulingPending, 1)
	require.False(t, s.schedulingPending[1].enabled)
	for i := int64(2); i <= 256; i++ {
		s.schedulingPending[i] = ticketSchedulingSignal{seenAt: time.Now()}
	}
	copy := *cfg
	copy.accountID = 257
	receipt.cfg = &copy
	s.queueTicketScheduling(receipt, false)
	require.Len(t, s.schedulingPending, 256)
	require.Equal(t, uint64(1), s.schedulingDropped.Load())
	_, err := applyTicketRulesPatch(ticketRulesFromConfig(cfg), &CodexTicketRulesPatch{DegradedSignalLength: ticketInt(292)})
	require.Error(t, err)
}
