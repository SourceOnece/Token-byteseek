//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/require"
	"strconv"
	"strings"
	"testing"
)

type ticketCreateRepo struct {
	*ticketHistoryStub
	settings *ticketSettingStub
	creates  int
	fail     bool
}

func (r *ticketCreateRepo) CreateWithCodexTicket(ctx context.Context, a *Account, groups []int64, in CodexTicketAccountCreation) error {
	if r.fail {
		return errors.New("synthetic failure")
	}
	r.creates++
	a.ID = int64(100 + r.creates)
	var cfg codexTicketConfig
	if err := json.Unmarshal([]byte(in.Configuration), &cfg); err != nil {
		return err
	}
	var override codexTicketAccountConfig
	if err := json.Unmarshal([]byte(in.AccountConfiguration), &override); err != nil {
		return err
	}
	if cfg.Accounts == nil {
		cfg.Accounts = map[string]codexTicketAccountConfig{}
	}
	cfg.Accounts[strconv.FormatInt(a.ID, 10)] = override
	raw, _ := json.Marshal(cfg)
	return r.settings.Set(ctx, codexTicketSettingsKey, string(raw))
}

// 默认值仅初始化新账号，覆盖、显式0、开关与代理秘密始终留在私有票据存储。
func TestCodexTicketImportDefaultsCreateOverrideAndFailure(t *testing.T) {
	s, base, _ := setupTicketManualTest(t, 292)
	repo := &ticketCreateRepo{ticketHistoryStub: base, settings: s.settings.(*ticketSettingStub)}
	s.gateway.accountRepo = repo
	mode, guard, length, zero, yes := "on", "recover_length", 332, 0, true
	before := s.config.Load().Generation
	template, err := s.UpdateImportDefaults(context.Background(), CodexTicketAccountPatch{Mode: &mode, WatchdogMode: &guard, VerifiedFlow: &yes, Rules: &CodexTicketRulesPatch{TargetLength: &length, MaxAttempts: &zero}}, nil)
	require.NoError(t, err)
	require.Equal(t, 332, template.Rules.TargetLength)
	require.Equal(t, 0, template.Rules.MaxAttempts)
	require.Equal(t, before, s.config.Load().Generation, "改模板不能废弃现有账号票据")
	require.Equal(t, 292, s.enabledAccountConfig(1).targetLength())
	a := ticketAccount()
	a.ID = 0
	done, err := s.CreateAccountWithDefaults(context.Background(), &a, nil, nil)
	require.NoError(t, err)
	require.True(t, done)
	require.Equal(t, 332, s.enabledAccountConfig(a.ID).targetLength())
	require.True(t, s.enabledAccountConfig(a.ID).VerifiedFlow)
	alternate := 356
	b := ticketAccount()
	b.ID = 0
	done, err = s.CreateAccountWithDefaults(context.Background(), &b, nil, &CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{TargetLength: &alternate}})
	require.NoError(t, err)
	require.True(t, done)
	require.Equal(t, 356, s.enabledAccountConfig(b.ID).targetLength())
	require.Equal(t, 332, s.enabledAccountConfig(a.ID).targetLength())
	repo.fail = true
	c := ticketAccount()
	c.ID = 0
	_, err = s.CreateAccountWithDefaults(context.Background(), &c, nil, nil)
	require.Error(t, err)
	require.Zero(t, c.ID)
	stale := "stale"
	_, err = s.UpdateImportDefaults(context.Background(), CodexTicketAccountPatch{Mode: &mode}, &stale)
	require.Error(t, err)
}

type ticketManagedProxyRepo struct {
	ProxyRepository
	proxy *Proxy
}

func (r *ticketManagedProxyRepo) GetByID(context.Context, int64) (*Proxy, error) {
	if r.proxy == nil {
		return nil, errors.New("missing")
	}
	return r.proxy, nil
}

func TestCodexTicketManagedProxyResolvesLiveAndNeverFallsBack(t *testing.T) {
	s, _, _ := setupTicketManualTest(t, 292)
	repo := &ticketManagedProxyRepo{proxy: &Proxy{ID: 9, Name: "managed", Protocol: "http", Host: "proxy.example", Port: 8080, Username: "user", Password: "synthetic-secret", Status: StatusActive}}
	s.proxyRepo = repo
	rows := []CodexTicketProxyUpdate{{Name: "chosen", ManagedID: 9}}
	cfg := s.config.Load()
	fixed := ""
	policy, err := s.updateTicketProxyPolicy(cfg, &CodexTicketProxyPolicyUpdate{Mode: "fixed", Proxies: &rows, FixedProxyID: &fixed})
	require.NoError(t, err)
	copy := *cfg
	copy.applyProxyPolicy(policy)
	p, ok := selectCodexTicketProxy(&copy, "", false)
	require.True(t, ok)
	require.Empty(t, p.Cipher)
	view, _ := json.Marshal(ticketProxyPolicyView(&copy))
	require.Contains(t, string(view), `"managed_proxy_id":9`)
	require.NotContains(t, string(view), "synthetic-secret")
	resolved, err := s.resolveTicketAttemptProxy(context.Background(), &copy, p)
	require.NoError(t, err)
	url, err := s.cipher.Decrypt(resolved.Cipher)
	require.NoError(t, err)
	require.True(t, strings.Contains(url, "proxy.example"))
	repo.proxy.Host = "changed.example"
	resolved, err = s.resolveTicketAttemptProxy(context.Background(), &copy, p)
	require.NoError(t, err)
	url, _ = s.cipher.Decrypt(resolved.Cipher)
	require.Contains(t, url, "changed.example")
	repo.proxy.Status = "inactive"
	_, err = s.resolveTicketAttemptProxy(context.Background(), &copy, p)
	require.Error(t, err)
	repo.proxy = nil
	_, err = s.resolveTicketAttemptProxy(context.Background(), &copy, p)
	require.Error(t, err)
}
