//go:build unit

package service

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/stretchr/testify/require"
)

// 与生产调度摘要一致，保留模型映射而省略令牌、工作区和组织；不得修改原账号映射。
func ticketSchedulerMetadata(account Account) Account {
	account.Credentials = map[string]any{"model_mapping": account.Credentials["model_mapping"]}
	return account
}

func TestCodexTicketReadyFromMetadataScheduler(t *testing.T) {
	for _, length := range []int{292, 332, 356} {
		for _, mode := range []string{"basic", "basic_batch", "advanced"} {
			t.Run(strconv.Itoa(length)+"/"+mode, func(t *testing.T) {
				s, repo, upstream := setupTicketManualTest(t, length)
				ctx := context.Background()
				groupID := int64(1)
				repo.accounts[0].GroupIDs = []int64{groupID}
				repo.accounts[0].Credentials["model_mapping"] = map[string]any{"alias": "gpt-6-astra", "gpt-6-astra": "gpt-6-astra", "gpt-5.6-sol": "gpt-5.6-sol"}
				batch, err := s.PrepareManualCollection(ctx, manualRequest(s))
				require.NoError(t, err)
				batch.Execute(func(string, any) bool { return true })
				status, err := s.Status(ctx, []int64{1})
				require.NoError(t, err)
				require.Equal(t, "ready", status.Items[0].Models[0].State)
				require.False(t, status.Items[0].Models[0].Blocked)
				full := repo.accounts[0]
				metadata := ticketSchedulerMetadata(full)
				cache := &openAISnapshotCacheStub{snapshotAccounts: []*Account{&metadata}, accountsByID: map[int64]*Account{1: &full}}
				g := s.gateway
				g.codexTickets.Store(s)
				g.cfg = &config.Config{}
				g.cfg.Gateway.Scheduling.LoadBatchEnabled = mode != "basic"
				g.cache = &schedulerTestGatewayCache{}
				g.concurrencyService = NewConcurrencyService(schedulerTestConcurrencyCache{})
				g.schedulerSnapshot = NewSchedulerSnapshotService(cache, nil, repo, nil, g.cfg)
				if mode == "advanced" {
					ctx = withAdvancedSchedulerTestGroup(ctx, groupID)
				}
				calls := upstream.calls.Load()
				for _, model := range []string{"gpt-6-astra", "gpt-5.6-sol", "alias"} {
					selection, _, err := g.SelectAccountWithScheduler(ctx, &groupID, "", "", model, nil, OpenAIUpstreamTransportAny, false)
					require.NoError(t, err, "有效票不能在摘要初筛时被排除")
					require.NotNil(t, selection)
					require.Equal(t, full.ID, selection.Account.ID)
					if selection.ReleaseFunc != nil {
						selection.ReleaseFunc()
					}
				}
				require.Equal(t, calls, upstream.calls.Load(), "调度补全不能发起采集")
				require.Empty(t, metadata.GetOpenAIAccessToken(), "共享摘要不能回填秘密")
				require.True(t, repo.accounts[0].Schedulable)
			})
		}
	}
}

type ticketMetadataCache struct {
	SchedulerCache
	account *Account
	err     error
	calls   int
	before  func(context.Context)
}

func (c *ticketMetadataCache) GetAccount(ctx context.Context, _ int64) (*Account, error) {
	c.calls++
	if c.before != nil {
		c.before(ctx)
	}
	return c.account, c.err
}

func TestCodexTicketMetadataLookupBoundaries(t *testing.T) {
	for _, name := range []string{"valid", "missing", "credential_changed", "workspace_changed", "organization_changed", "different_account", "api_key", "missing_token", "cache_error", "timeout", "disabled_during_read", "generation_changed"} {
		t.Run(name, func(t *testing.T) {
			s, _, _ := newTicketTestService()
			enableTicketTest(t, s)
			ctx := context.Background()
			full := ticketAccount()
			metadata := ticketSchedulerMetadata(full)
			seedTicket(t, s, &full, "gpt-6-astra", "fake-token")
			cache := &ticketMetadataCache{account: &full}
			// 空仓储明确返回not found，缓存故障不借其它账号或旧票放行。
			s.gateway = &OpenAIGatewayService{schedulerSnapshot: NewSchedulerSnapshotService(cache, nil, &ticketAccountStub{}, nil, nil)}
			wantBlocked := name != "valid" && name != "disabled_during_read"
			switch name {
			case "missing":
				s.cache.(*ticketCacheStub).values = map[string]string{}
			case "credential_changed":
				full.Credentials["access_token"] = "new-token"
			case "workspace_changed":
				full.Credentials["chatgpt_account_id"] = "workspace-b"
			case "organization_changed":
				full.Credentials["organization_id"] = "organization-b"
			case "different_account":
				full.ID = 2
			case "api_key":
				full.Type = AccountTypeAPIKey
			case "missing_token":
				delete(full.Credentials, "access_token")
			case "cache_error":
				cache.account = nil
				cache.err = errors.New("offline")
			case "timeout":
				cache.before = func(ctx context.Context) { <-ctx.Done() }
			case "disabled_during_read":
				cache.before = func(context.Context) {
					no := false
					_, err := s.Update(ctx, CodexTicketSettingsUpdate{Enabled: &no})
					require.NoError(t, err)
				}
			case "generation_changed":
				cache.before = func(context.Context) { _, err := s.Update(ctx, CodexTicketSettingsUpdate{}); require.NoError(t, err) }
			}
			start := time.Now()
			require.Equal(t, wantBlocked, s.Blocks(ctx, &metadata, "gpt-6-astra"))
			require.Less(t, time.Since(start), time.Second)
			require.Equal(t, 1, cache.calls)
			require.Empty(t, metadata.GetOpenAIAccessToken())
		})
	}
}

func TestCodexTicketMetadataNoExtraReadsOutsideGate(t *testing.T) {
	s, _, _ := newTicketTestService()
	enableTicketTest(t, s)
	ctx := context.Background()
	full := ticketAccount()
	metadata := ticketSchedulerMetadata(full)
	cache := &ticketMetadataCache{account: &full}
	s.gateway = &OpenAIGatewayService{schedulerSnapshot: NewSchedulerSnapshotService(cache, nil, &ticketAccountStub{}, nil, nil)}
	seedTicket(t, s, &full, "gpt-6-astra", "fake-token")
	require.False(t, s.Blocks(ctx, &full, "gpt-6-astra"))
	require.False(t, s.Blocks(ctx, &metadata, "not-target"))
	api := metadata
	api.Type = AccountTypeAPIKey
	require.False(t, s.Blocks(ctx, &api, "gpt-6-astra"))
	require.False(t, s.Blocks(ctx, nil, "gpt-6-astra"))
	no := false
	_, err := s.Update(ctx, CodexTicketSettingsUpdate{Enabled: &no})
	require.NoError(t, err)
	require.False(t, s.Blocks(ctx, &metadata, "gpt-6-astra"))
	require.Zero(t, cache.calls)
}

func TestCodexTicketMetadataRepositoryFallbackAndFinalIdentity(t *testing.T) {
	s, _, _ := newTicketTestService()
	enableTicketTest(t, s)
	ctx := context.Background()
	full := ticketAccount()
	metadata := ticketSchedulerMetadata(full)
	s.gateway = &OpenAIGatewayService{accountRepo: &ticketAccountStub{accounts: []Account{full}}}
	seedTicket(t, s, &full, "gpt-6-astra", "fake-token")
	require.False(t, s.Blocks(ctx, &metadata, "gpt-6-astra"))
	// 最终注入仍按本次真实Bearer复核，初筛成功不能借用旧凭据票。
	headers := http.Header{"Authorization": {"Bearer rotated-token"}}
	require.ErrorIs(t, s.Apply(ctx, &full, "gpt-6-astra", headers), ErrCodexTicketUnavailable)
	require.Empty(t, headers.Get(openAICodexTurnStateHeader))
}

// 同池第一号缺票时必须选第二号的同模型票；不能借别号或另一模型的票放行。
func TestCodexTicketMetadataAccountAndModelIsolation(t *testing.T) {
	s, _, _ := newTicketTestService()
	enableTicketTest(t, s)
	ctx := context.Background()
	first := ticketAccount()
	second := ticketAccount()
	second.ID = 2
	second.Credentials["access_token"] = "second-token"
	first.GroupIDs = []int64{1}
	second.GroupIDs = []int64{1}
	firstMeta, secondMeta := ticketSchedulerMetadata(first), ticketSchedulerMetadata(second)
	cache := &openAISnapshotCacheStub{snapshotAccounts: []*Account{&firstMeta, &secondMeta}, accountsByID: map[int64]*Account{1: &first, 2: &second}}
	repo := &ticketAccountStub{accounts: []Account{first, second}}
	g := &OpenAIGatewayService{accountRepo: repo, cache: &schedulerTestGatewayCache{}, concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{})}
	g.schedulerSnapshot = NewSchedulerSnapshotService(cache, nil, repo, nil, nil)
	g.codexTickets.Store(s)
	s.gateway = g
	seedTicket(t, s, &second, "gpt-6-astra", "second-token")
	require.True(t, s.Blocks(ctx, &firstMeta, "gpt-6-astra"))
	require.False(t, s.Blocks(ctx, &secondMeta, "gpt-6-astra"))
	require.True(t, s.Blocks(ctx, &secondMeta, "gpt-5.6-sol"))
	groupID := int64(1)
	selection, err := g.SelectAccountWithLoadAwareness(ctx, &groupID, "", "gpt-6-astra", nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), selection.Account.ID)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
	require.True(t, repo.accounts[0].Schedulable)
	require.True(t, repo.accounts[1].Schedulable)
}
