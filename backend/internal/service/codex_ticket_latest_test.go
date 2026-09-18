//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 最新失败不销毁旧票、改门控；迟到旧完成记录不能盖住新结果。
func TestCodexTicketLatestManualFailureKeepsValidTicket(t *testing.T) {
	s, r, u := setupTicketManualTest(t, 292)
	a := r.accounts[0]
	seedTicket(t, s, &a, "gpt-6-astra", "fake-token")
	u.ticket = "gAAAAA" + strings.Repeat("x", 306)
	m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
	require.NoError(t, err)
	m.Execute(func(string, any) bool { return true })
	result, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	row := result.Items[0].Models[0]
	require.Equal(t, "ready", row.State)
	require.False(t, row.Blocked)
	require.NotNil(t, row.Latest)
	require.Equal(t, "manual", row.Latest.Source)
	require.Equal(t, "missing", row.Latest.State)
	require.Equal(t, 312, row.Latest.Diagnostic.HeaderLength)
	key := codexTicketKey(s.config.Load(), &a, "gpt-6-astra", "fake-token")
	s.recordLatest(context.Background(), s.config.Load(), key, "auto", CodexTicketAttempt{Status: "ready", FinishedAt: row.Latest.CheckedAt.Add(-time.Second)})
	result, err = s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, "manual", result.Items[0].Models[0].Latest.Source)
	s.recordLatest(context.Background(), s.config.Load(), key, "auto", CodexTicketAttempt{Status: "failed", FinishedAt: row.Latest.CheckedAt.Add(time.Second)})
	result, err = s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, "auto", result.Items[0].Models[0].Latest.Source)
	require.False(t, s.Blocks(context.Background(), &a, "gpt-6-astra"))
}
func TestCodexTicketLatestSanitizationAndIsolation(t *testing.T) {
	s, r, _ := setupTicketManualTest(t, 292)
	a := r.accounts[0]
	key := codexTicketKey(s.config.Load(), &a, "gpt-6-astra", "fake-token")
	s.recordLatest(context.Background(), s.config.Load(), key, "manual", CodexTicketAttempt{Status: "failed", Reason: "secret", FinishedAt: time.Now(), ReferenceIP: "https://secret.invalid", IPStatus: "secret", IPSource: "secret", IPHTTPStatus: 900})
	result, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	raw, _ := json.Marshal(result)
	require.NotContains(t, string(raw), "secret")
	yes := true
	_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{Enabled: &yes})
	require.NoError(t, err)
	result, err = s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Nil(t, result.Items[0].Models[0].Latest)
}

type ticketReferenceDiagnosticStub struct{ ticketIPStub }

func (p *ticketReferenceDiagnosticStub) ProbeTicketReferenceIP(context.Context, string) CodexTicketReferenceIP {
	return CodexTicketReferenceIP{Status: "http_error", Source: "chatgpt_trace", HTTPStatus: 403}
}
func TestCodexTicketManualReferenceFailureDoesNotChangeTicket(t *testing.T) {
	s, r, u := setupTicketManualTest(t, 292)
	s.ipProber = &ticketReferenceDiagnosticStub{}
	m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
	require.NoError(t, err)
	m.Execute(func(string, any) bool { return true })
	require.Equal(t, int32(2), u.calls.Load())
	require.Equal(t, 2, r.run.Counts["ready"])
	for _, event := range r.events {
		require.Equal(t, "ready", event.Status)
		require.Equal(t, "http_error", event.IPStatus)
		require.Equal(t, 403, event.IPHTTPStatus)
		require.Empty(t, event.ReferenceIP)
	}
	status, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, "manual", status.Items[0].Models[0].Latest.Source)
	require.Equal(t, "http_error", status.Items[0].Models[0].Latest.IPStatus)
	require.False(t, status.Items[0].Models[0].Blocked)
}
