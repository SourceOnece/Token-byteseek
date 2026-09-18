//go:build unit

package service

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 默认兼容，去重/非法值拦截；信号和模型保存不影响已有代理密码回显边界。
func TestCodexTicketModelsAndSignalSettings(t *testing.T) {
	s, _, _ := newTicketTestService()
	view, err := s.View(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-6-astra", "gpt-5.6-sol"}, view.Models)
	require.Zero(t, view.DegradedSignalLength)
	enableTicketTest(t, s)
	models := []string{" gpt-6-astra ", "custom-codex-model", "custom-codex-model"}
	signal := 312
	view, err = s.Update(context.Background(), CodexTicketSettingsUpdate{Models: &models, DegradedSignalLength: &signal})
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-6-astra", "custom-codex-model"}, view.Models)
	require.Equal(t, 312, view.DegradedSignalLength)
	for _, invalid := range [][]string{{}, {"with space"}, {"bad\nmodel"}, {strings.Repeat("a", 257)}, make([]string, 101)} {
		_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{Models: &invalid})
		require.Error(t, err)
	}
	for _, invalid := range []int{-1, 5, 8193} {
		_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{DegradedSignalLength: &invalid})
		require.Error(t, err)
	}
	zero := 0
	view, err = s.Update(context.Background(), CodexTicketSettingsUpdate{DegradedSignalLength: &zero})
	require.NoError(t, err)
	require.Zero(t, view.DegradedSignalLength)
}

func TestCodexTicketConfiguredModelsManualProgressAndStatus(t *testing.T) {
	s, r, u := setupTicketManualTest(t, 292)
	models := []string{"gpt-6-astra", "gpt-5.6-sol", "custom-codex-model"}
	_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{Models: &models})
	require.NoError(t, err)
	m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
	require.NoError(t, err)
	require.Equal(t, 3, m.Run.Total)
	m.Execute(func(string, any) bool { return true })
	require.Equal(t, int32(3), u.calls.Load())
	require.Equal(t, 3, r.run.Counts["ready"])
	status, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, models, status.Models)
	require.Len(t, status.Items[0].Models, 3)
	require.Equal(t, "custom-codex-model", status.Items[0].Models[2].Model)
	require.Equal(t, "ready", status.Items[0].Models[2].State)
	a := r.accounts[0]
	require.False(t, s.Blocks(context.Background(), &a, "custom-codex-model"))
	other := a
	other.ID = 2
	require.True(t, s.Blocks(context.Background(), &other, "custom-codex-model"))
	require.False(t, s.Blocks(context.Background(), &other, "not-configured"))
	headers := http.Header{"Authorization": {"Bearer fake-token"}}
	require.NoError(t, s.Apply(context.Background(), &a, "custom-codex-model", headers))
	require.Len(t, headers.Get(openAICodexTurnStateHeader), 292)
}

func TestCodexTicketConfiguredModelsAutomaticHarvest(t *testing.T) {
	s, r, u := setupTicketManualTest(t, 292)
	models := []string{"custom-codex-one", "custom-codex-two", "custom-codex-three"}
	_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{Models: &models})
	require.NoError(t, err)
	s.harvest(context.Background())
	require.Equal(t, int32(3), u.calls.Load())
	status, err := s.Status(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Len(t, status.Items[0].Models, 3)
	for _, model := range status.Items[0].Models {
		require.Equal(t, "ready", model.State)
		require.Equal(t, "auto", model.Latest.Source)
	}
	require.True(t, r.accounts[0].Schedulable)
}

// 即使信号长度与合格长度相同也只标记提示，不能新增废票/停调副作用。
func TestCodexTicketDegradedSignalIsDiagnosticOnly(t *testing.T) {
	for _, length := range []int{292, 312, 356} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			s, r, u := setupTicketManualTest(t, 292)
			signal := length
			_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{DegradedSignalLength: &signal})
			require.NoError(t, err)
			u.ticket = "gAAAAA" + strings.Repeat("x", length-6)
			m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
			require.NoError(t, err)
			m.Execute(func(string, any) bool { return true })
			for _, event := range r.events {
				require.True(t, event.Diagnostic.DegradedSignal)
				if length == 292 {
					require.Equal(t, "ready", event.Status)
				} else {
					require.Equal(t, "missing", event.Status)
				}
			}
			require.True(t, r.accounts[0].Schedulable)
		})
	}
}

func TestCodexTicketDegradedSignalDisabledAndNonMatch(t *testing.T) {
	for _, signal := range []int{0, 356} {
		s, r, _ := setupTicketManualTest(t, 292)
		_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{DegradedSignalLength: &signal})
		require.NoError(t, err)
		m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
		require.NoError(t, err)
		m.Execute(func(string, any) bool { return true })
		for _, event := range r.events {
			require.False(t, event.Diagnostic.DegradedSignal)
			require.Equal(t, "ready", event.Status)
		}
	}
}
