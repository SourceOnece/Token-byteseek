//go:build unit

package service

import (
	"context"
	"encoding/json"
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
	err := configureTicketTestAccount(t, s, CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{Models: &models}})
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
	require.False(t, s.Blocks(context.Background(), &other, "custom-codex-model"), "另一旧号未配置该模型，不因一号配置新增门控")
	require.False(t, s.Blocks(context.Background(), &other, "not-configured"))
	headers := http.Header{"Authorization": {"Bearer fake-token"}}
	require.NoError(t, s.Apply(context.Background(), &a, "custom-codex-model", headers))
	require.Len(t, headers.Get(openAICodexTurnStateHeader), 292)
}

func TestCodexTicketConfiguredModelsAutomaticHarvest(t *testing.T) {
	s, r, u := setupTicketManualTest(t, 292)
	models := []string{"custom-codex-one", "custom-codex-two", "custom-codex-three"}
	err := configureTicketTestAccount(t, s, CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{Models: &models}})
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

// 新规则明确命中降智长度就关调度，历史长度冲突也优先按异常，不同轮先关再开。
func TestCodexTicketDegradedSignalClosesScheduling(t *testing.T) {
	for _, length := range []int{292, 312, 356} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			s, r, u := setupTicketManualTest(t, 292)
			s.gateway.accountRepo = &ticketSchedulingRepo{ticketHistoryStub: r}
			signal := length
			if signal == 292 {
				// 明确构造升级前两长度冲突的历史数据；现行账号和模板写入均应拒绝它。
				cfg := *s.config.Load()
				baseline := *cfg.LegacyAccountDefaults
				rules := *baseline.Rules
				rules.DegradedSignalLength = signal
				baseline.Rules = &rules
				cfg.LegacyAccountDefaults = &baseline
				raw, e := json.Marshal(&cfg)
				require.NoError(t, e)
				require.NoError(t, s.settings.Set(context.Background(), codexTicketSettingsKey, string(raw)))
				s.config.Store(&cfg)
			} else {
				require.NoError(t, configureTicketTestAccount(t, s, CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{DegradedSignalLength: &signal}}))
			}
			u.ticket = "gAAAAA" + strings.Repeat("x", length-6)
			m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
			require.NoError(t, err)
			m.Execute(func(string, any) bool { return true })
			for _, event := range r.events {
				require.True(t, event.Diagnostic.DegradedSignal)
				require.Equal(t, "missing", event.Status)
			}
			require.False(t, r.accounts[0].Schedulable)
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
