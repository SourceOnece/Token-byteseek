//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 构造升级前磁盘JSON，不能通过新写接口模拟旧数据，否则会丢失迁移边界。
func TestCodexTicketNormalizationPreservesEffectiveRulesKeysAndProxy(t *testing.T) {
	s, _, settings := newTicketTestService()
	proxy, _ := s.cipher.Encrypt("http://user:synthetic@old.invalid:8080")
	legacy := &codexTicketConfig{Enabled: true, Generation: "global-before", WatchdogMode: "recover_length", TargetLength: 332, MaxAttempts: 0, Models: []string{"model-a"}, ProxyCipher: proxy, Accounts: map[string]codexTicketAccountConfig{
		"1": {Mode: "inherit", WatchdogMode: "inherit", Revision: "one", VerifiedFlow: true, ProxyCipher: proxy},
		"2": {Mode: "off", WatchdogMode: "off", Revision: "two"},
	}}
	raw, err := json.Marshal(legacy)
	require.NoError(t, err)
	require.NoError(t, settings.Set(context.Background(), codexTicketSettingsKey, string(raw)))
	a := ticketAccount()
	// 独立重建旧键输入，不能用新配置解析器计算两遍来冒充兼容性证明。
	oldEffective := *legacy
	oldEffective.Generation += ":one"
	oldEffective.VerifiedFlow = true
	keyBefore := codexTicketKey(&oldEffective, &a, "model-a", "fake-token")
	cfg, err := s.readConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, "global-before", cfg.Generation)
	require.Equal(t, "one", cfg.Accounts["1"].Revision)
	require.Equal(t, "on", cfg.Accounts["1"].Mode)
	require.Equal(t, "recover_length", cfg.Accounts["1"].WatchdogMode)
	require.Equal(t, "recover", ticketConfigForAccount(cfg, 1).WatchdogMode)
	require.Equal(t, 332, cfg.Accounts["1"].Rules.TargetLength)
	require.Equal(t, 1, cfg.Accounts["1"].Rules.MaxAttempts, "历史全局0不能变无限")
	require.Empty(t, cfg.Accounts["1"].ProxyCipher)
	require.Equal(t, proxy, cfg.Accounts["1"].ProxyPolicy.Proxies[0].Cipher)
	require.Equal(t, keyBefore, codexTicketKey(ticketConfigForAccount(cfg, 1), &a, "model-a", "fake-token"))
	require.False(t, ticketConfigForAccount(cfg, 2).Enabled)
	require.Equal(t, 332, ticketConfigForAccount(cfg, 99).targetLength(), "无独立旧账号仍取固化历史值")
	stored, _ := settings.GetValue(context.Background(), codexTicketSettingsKey)
	require.Equal(t, string(raw), stored, "只读归一不能写数据库")
	// 内部重复归一/序列化不能改变密文、账号版本、旧0含义或备用守护。
	first, _ := json.Marshal(cfg)
	normalizeTicketConfiguration(cfg)
	second, _ := json.Marshal(cfg)
	require.Equal(t, first, second)
	legacy.Enabled = false
	account := ticketAccountSettingsView(legacy, 1)
	require.Equal(t, "on", account.Mode)
	require.False(t, account.EffectiveEnabled)
}

func TestCodexTicketNormalizationRetainsPreviouslyCachedTicket(t *testing.T) {
	s, cache, settings := newTicketTestService()
	a := ticketAccount()
	s.gateway = &OpenAIGatewayService{accountRepo: &ticketAccountStub{accounts: []Account{a}}}
	legacy := &codexTicketConfig{Enabled: true, Generation: "same-generation", TargetLength: 332, Models: []string{"gpt-6-astra"}}
	raw, err := json.Marshal(legacy)
	require.NoError(t, err)
	require.NoError(t, settings.Set(context.Background(), codexTicketSettingsKey, string(raw)))
	oldKey := codexTicketKey(legacy, &a, "gpt-6-astra", a.GetOpenAIAccessToken())
	value := codexTicketValue{State: "gAAAAA" + strings.Repeat("x", 326), ExpiresAt: time.Now().Add(time.Hour)}
	body, _ := json.Marshal(value)
	encrypted, _ := s.cipher.Encrypt(string(body))
	require.NoError(t, cache.Set(context.Background(), oldKey, encrypted, time.Hour))
	cfg, err := s.readConfig(context.Background())
	require.NoError(t, err)
	s.config.Store(cfg)
	read, ok := s.lookup(context.Background(), s.enabledAccountConfig(1), &a, "gpt-6-astra", a.GetOpenAIAccessToken())
	require.True(t, ok)
	require.Equal(t, value.State, read.State)
	length := 356
	_, err = s.UpdateImportDefaults(context.Background(), CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{TargetLength: &length}}, nil)
	require.NoError(t, err)
	_, ok = s.lookup(context.Background(), s.enabledAccountConfig(1), &a, "gpt-6-astra", a.GetOpenAIAccessToken())
	require.True(t, ok, "改新号模板也不使升级前旧号缓存失效")
}

func TestCodexTicketNewDefaultAndExistingAccountAreIndependent(t *testing.T) {
	s, base, _ := setupTicketManualTest(t, 332)
	repo := &ticketCreateRepo{ticketHistoryStub: base, settings: s.settings.(*ticketSettingStub)}
	s.gateway.accountRepo = repo
	cfg := s.config.Load()
	old := ticketConfigForAccount(cfg, 1)
	a, _ := repo.GetByID(context.Background(), 1)
	oldKey := codexTicketKey(old, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	length, zero, guard := 356, 0, "recover_model"
	_, err := s.UpdateImportDefaults(context.Background(), CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{TargetLength: &length, MaxAttempts: &zero}, WatchdogMode: &guard}, nil)
	require.NoError(t, err)
	created := ticketAccount()
	created.ID = 0
	done, err := s.CreateAccountWithDefaults(context.Background(), &created, nil, nil)
	require.NoError(t, err)
	require.True(t, done)
	current := s.enabledAccountConfig(created.ID)
	require.Equal(t, 356, current.targetLength())
	require.Zero(t, current.attempts())
	require.Equal(t, "recover_model", current.WatchdogMode)
	require.Equal(t, 332, s.enabledAccountConfig(1).targetLength())
	require.Equal(t, oldKey, codexTicketKey(s.enabledAccountConfig(1), a, "gpt-6-astra", a.GetOpenAIAccessToken()))
	length = 512
	_, err = s.UpdateImportDefaults(context.Background(), CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{TargetLength: &length}}, nil)
	require.NoError(t, err)
	require.Equal(t, 356, s.enabledAccountConfig(created.ID).targetLength(), "改模板不追改新建后的账号")
	// 旧网关字段转写同一导入模板，不修改原账号和票据代次。
	beforeGen := s.config.Load().Generation
	length = 444
	view, err := s.Update(context.Background(), CodexTicketSettingsUpdate{TargetLength: &length})
	require.NoError(t, err)
	require.Equal(t, 444, view.TargetLength)
	require.Equal(t, beforeGen, s.config.Load().Generation)
	defaults, err := s.ImportDefaults(context.Background())
	require.NoError(t, err)
	require.Equal(t, 444, defaults.Rules.TargetLength)
	require.Equal(t, 332, s.enabledAccountConfig(1).targetLength())
	require.Equal(t, 356, s.enabledAccountConfig(created.ID).targetLength())
}

func TestCodexTicketCreateWithoutSavedTemplateStillFreezesDefaults(t *testing.T) {
	s, base, _ := setupTicketManualTest(t, 292)
	repo := &ticketCreateRepo{ticketHistoryStub: base, settings: s.settings.(*ticketSettingStub)}
	s.gateway.accountRepo = repo
	created := ticketAccount()
	created.ID = 0
	done, err := s.CreateAccountWithDefaults(context.Background(), &created, nil, nil)
	require.NoError(t, err)
	require.True(t, done)
	stored := s.config.Load().Accounts[strconv.FormatInt(created.ID, 10)]
	require.NotNil(t, stored.Rules)
	require.Equal(t, "on", stored.Mode)
	require.Equal(t, "observe", stored.WatchdogMode)
}

func TestCodexTicketExplicitUnlimitedAndBackupGuardSurviveNormalization(t *testing.T) {
	zero := 0
	root := &codexTicketConfig{WatchdogMode: "off", Enabled: true}
	rules, _ := applyTicketRulesPatch(ticketRulesFromConfig(root), &CodexTicketRulesPatch{MaxAttempts: &zero})
	root.Accounts = map[string]codexTicketAccountConfig{"1": {Rules: &rules, Mode: "on", WatchdogMode: "inherit", VerifiedFlow: true}}
	normalizeTicketConfiguration(root)
	require.Equal(t, "off", root.Accounts["1"].WatchdogMode)
	require.Zero(t, ticketConfigForAccount(root, 1).attempts())
	account := root.Accounts["1"]
	account.VerifiedFlow = false
	root.Accounts["1"] = account
	require.Equal(t, "off", ticketConfigForAccount(root, 1).WatchdogMode)
}

func TestCodexTicketStageLengthsNeverTurnInvalidIntoEmpty(t *testing.T) {
	for _, length := range []int{-10, 0, 332, 8193, 65535} {
		safe := safeCodexTicketDiagnostic(&CodexTicketDiagnostic{Stages: []CodexTicketValidationStage{{Name: "verify", StateLength: length, HTTPStatus: 200}}})
		if length < 0 {
			require.Equal(t, -1, safe.Stages[0].StateLength)
		} else {
			require.Equal(t, length, safe.Stages[0].StateLength)
		}
	}
}

func TestCodexTicketOldProxyWritesStayInCanonicalPolicy(t *testing.T) {
	s, _, a := setupTicketProxyTest(t, "fixed", 1)
	address := "http://user:synthetic@changed.invalid:8080"
	_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{HarvestProxyURL: &address})
	require.Error(t, err, "旧单地址不能覆盖多代理")
	rows := []CodexTicketProxyUpdate{{ID: "legacy", Name: "A", URL: address}}
	_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{Proxies: &rows})
	require.NoError(t, err)
	config := s.enabledAccountConfig(a.ID)
	raw, err := s.cipher.Decrypt(config.ProxyPolicy.Proxies[0].Cipher)
	require.NoError(t, err)
	require.Equal(t, address, raw)
}
