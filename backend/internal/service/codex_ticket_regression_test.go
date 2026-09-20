//go:build unit

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 仅用合成账号重现：旧事件已出队，新事件在旧事件写库前到达。
func TestCodexTicketRegressionSchedulingDoesNotDiscardNewerBusinessSignal(t *testing.T) {
	for _, initial := range []bool{false, true} {
		t.Run(fmt.Sprint(initial), func(t *testing.T) {
			s, base, _ := setupTicketManualTest(t, 292)
			base.accounts[0].Schedulable = initial
			r := &ticketSchedulingRepo{ticketHistoryStub: base}
			s.gateway.accountRepo = r
			a, err := r.GetByID(context.Background(), 1)
			require.NoError(t, err)
			cfg := s.enabledAccountConfig(1)
			key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
			receipt := &codexTicketReceipt{s: s, cfg: cfg, key: key, encoded: "audit-encrypted-ticket", model: "gpt-6-astra"}
			require.NoError(t, s.cache.Set(context.Background(), key, receipt.encoded, time.Hour))
			s.queueTicketScheduling(receipt, !initial)
			older := s.schedulingPending[1]
			delete(s.schedulingPending, 1)
			s.queueTicketScheduling(receipt, initial)
			s.applyTicketSchedulingSignal(context.Background(), older)
			newer := s.schedulingPending[1]
			s.applyTicketSchedulingSignal(context.Background(), newer)
			a, err = r.GetByID(context.Background(), 1)
			require.NoError(t, err)
			t.Logf("初始调度=%v，先收到=%v，后收到=%v，实际最终=%v，写库次数=%d", initial, !initial, initial, a.Schedulable, r.writes)
			require.Equal(t, initial, a.Schedulable, "较新的业务信号不应被本工作线程自己的较早更新丢弃")
		})
	}
}

// 无限手动任务让出工作线程后，其他账号应仍能自动采票。
func TestCodexTicketRegressionManualRetryDoesNotStopOtherAccountMaintenance(t *testing.T) {
	s, r, u := setupTicketManualTest(t, 292)
	second := ticketAccount()
	second.ID = 2
	r.accounts = append(r.accounts, second)
	models, unlimited := []string{"gpt-6-astra"}, 0
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{Models: &models, MaxAttempts: &unlimited}}})
	require.NoError(t, err)
	u.ticket = "gAAAAA" + strings.Repeat("x", 350)
	m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
	require.NoError(t, err)
	defer m.Close()
	task := &codexTicketManualTask{id: 1, model: "gpt-6-astra"}
	m.collectModel(context.Background(), 1, task.model, func(CodexTicketAttempt) {}, task)
	require.False(t, task.done)
	require.Equal(t, 1, task.attempt)
	before := u.calls.Load()
	// 直接调用后台轮次，排除本机入口拦截后仍被全局租约挡住。
	s.harvest(context.Background())
	t.Logf("手动账号1已让出线程；后台账号2有资格且缺票；自动新增请求数=%d", u.calls.Load()-before)
	require.Greater(t, u.calls.Load(), before, "手动等待重试不应冻结未选中账号的续票")
}

// 复现真实冷却缓存的关键行为：首次失败后写入未来冷却时间。
type ticketRegressionCooldownCache struct{ *ticketCacheStub }

func (c *ticketRegressionCooldownCache) RecordTicketCollection(ctx context.Context, key string, success bool, threshold, seconds, maximum int, now time.Time) error {
	if !success && threshold > 0 {
		return c.Set(ctx, "collection:"+key, fmt.Sprintf(`{"failures":1,"until":%d}`, now.Add(time.Duration(seconds)*time.Second).UnixMilli()), time.Hour)
	}
	return nil
}

func TestCodexTicketRegressionCooldownDoesNotInventNetworkAttempts(t *testing.T) {
	s, r, u := setupTicketManualTest(t, 292)
	s.cache = &ticketRegressionCooldownCache{s.cache.(*ticketCacheStub)}
	max, threshold, seconds := 3, 1, 60
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{MaxAttempts: &max, FailureThreshold: &threshold, CooldownSeconds: &seconds}}})
	require.NoError(t, err)
	u.ticket = "gAAAAA" + strings.Repeat("x", 350)
	a, _ := r.GetByID(context.Background(), 1)
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	cfg := s.enabledAccountConfig(1)
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	raw, err := s.cache.Get(context.Background(), "latest:"+key)
	require.NoError(t, err)
	var latest CodexTicketLatest
	require.NoError(t, json.Unmarshal([]byte(raw), &latest))
	require.NotNil(t, latest.Diagnostic)
	t.Logf("实际请求=%d，最新日志次数=%d，结果=%s/%s，长度=%d", u.calls.Load(), latest.Diagnostic.Attempt, latest.State, latest.Reason, latest.Diagnostic.HeaderLength)
	require.Equal(t, int32(1), u.calls.Load())
	require.Equal(t, 1, latest.Diagnostic.Attempt, "未发请求的冷却检查不能增加采集轮数或抹掉上次长度")
}

// HTTP已成功建流，但流内明确给出鉴权、限流或额度失败。
func TestCodexTicketRegressionVerifiedFlowPreservesStreamErrorBackoff(t *testing.T) {
	for _, code := range []string{"insufficient_quota", "rate_limit_exceeded", "invalid_api_key"} {
		t.Run(code, func(t *testing.T) {
			s, r, _ := setupVerifiedTicketTest(t, true)
			a, _ := r.GetByID(context.Background(), 1)
			cfg := s.enabledAccountConfig(1)
			body := `data: {"type":"error","error":{"code":"` + code + `","message":"synthetic rejection"}}` + "\n\n"
			response := verifiedResponse(200, 292, "gpt-6-astra")
			response.Body = io.NopCloser(strings.NewReader(body))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			diagnostic := &CodexTicketDiagnostic{}
			ready, retry, reason := s.validateTicketChain(ctx, cancel, cfg, a, "gpt-6-astra", nil, nil, response, diagnostic)
			t.Logf("错误=%s，ready=%v，retry=%v，reason=%s，分类=%q，退避=%v", code, ready, retry, reason, diagnostic.ErrorKind, diagnostic.RetryNotBefore)
			require.False(t, ready)
			require.False(t, retry, "流内的明确拒绝应停止当前重试并保留退避保护")
		})
	}
	completion := parseTicketCompletion(strings.NewReader(`data: {"type":"response.failed","response":{"error":{"type":"insufficient_quota"}}}`+"\n\n"), "gpt-6-astra")
	require.Equal(t, "quota", completion.ErrorKind, "兼容response.error.type错误信封")
}

// 真实响应之后访问数据库和发布缓存仍须带截止时间，不能越过短并发租约。
type ticketBoundedReadRepo struct {
	*ticketSchedulingRepo
	t             *testing.T
	afterResponse bool
}

func (r *ticketBoundedReadRepo) GetByID(ctx context.Context, id int64) (*Account, error) {
	if r.afterResponse {
		deadline, ok := ctx.Deadline()
		require.True(r.t, ok)
		require.LessOrEqual(r.t, time.Until(deadline), 2*time.Second)
	}
	return r.ticketSchedulingRepo.GetByID(ctx, id)
}

type ticketBoundedWriteCache struct {
	*ticketCacheStub
	t *testing.T
}

func (c *ticketBoundedWriteCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if len(key) == 64 {
		deadline, ok := ctx.Deadline()
		require.True(c.t, ok)
		require.LessOrEqual(c.t, time.Until(deadline), time.Second)
	}
	return c.ticketCacheStub.Set(ctx, key, value, ttl)
}

func TestCodexTicketRegressionPublicationIOBounded(t *testing.T) {
	s, base, u := setupTicketManualTest(t, 292)
	r := &ticketBoundedReadRepo{ticketSchedulingRepo: &ticketSchedulingRepo{ticketHistoryStub: base}, t: t}
	s.gateway.accountRepo = r
	s.cache = &ticketBoundedWriteCache{ticketCacheStub: s.cache.(*ticketCacheStub), t: t}
	u.before = func(*http.Request) { r.afterResponse = true }
	a, _ := r.GetByID(context.Background(), 1)
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	_, ready := s.lookup(context.Background(), s.enabledAccountConfig(1), a, "gpt-6-astra", a.GetOpenAIAccessToken())
	require.True(t, ready)
}

// 配置暂时读不到与管理员主动关闭开关应是不同状态。
func TestCodexTicketRegressionStaleSettingsDoNotSilentlyBypassVerifiedFlow(t *testing.T) {
	s, r, _ := setupVerifiedTicketTest(t, true)
	a, _ := r.GetByID(context.Background(), 1)
	require.True(t, s.Blocks(context.Background(), a, "gpt-6-astra"))
	cfg := *s.config.Load()
	cfg.loadedAt = time.Now().Add(-11 * time.Second)
	s.config.Store(&cfg)
	headers := http.Header{"Authorization": {"Bearer " + a.GetOpenAIAccessToken()}}
	err := s.Apply(context.Background(), a, "gpt-6-astra", headers)
	t.Logf("管理员开关仍=%v；票据缺失；配置过期后Blocks=%v、Apply错误=%v、注入STATE=%v", cfg.Enabled, s.Blocks(context.Background(), a, "gpt-6-astra"), err, headers.Get(openAICodexTurnStateHeader) != "")
	require.Error(t, err, "配置短暂失联不应等同于管理员主动关闭双链路验证")
}

// 精确放行本worker的旧写入，不能顺带放行管理员/质量检测之后产生的新版本。
func TestCodexTicketRegressionSchedulingStillProtectsExternalChanges(t *testing.T) {
	s, base, _ := setupTicketManualTest(t, 292)
	r := &ticketSchedulingRepo{ticketHistoryStub: base}
	s.gateway.accountRepo = r
	a, _ := r.GetByID(context.Background(), 1)
	cfg := s.enabledAccountConfig(1)
	key := codexTicketKey(cfg, a, "gpt-6-astra", a.GetOpenAIAccessToken())
	receipt := &codexTicketReceipt{s: s, cfg: cfg, key: key, encoded: "audit-ticket", model: "gpt-6-astra"}
	require.NoError(t, s.cache.Set(context.Background(), key, receipt.encoded, time.Hour))
	s.queueTicketScheduling(receipt, false)
	older := s.schedulingPending[1]
	delete(s.schedulingPending, 1)
	s.queueTicketScheduling(receipt, true)
	s.applyTicketSchedulingSignal(context.Background(), older)
	require.False(t, s.schedulingPending[1].previousWrite.IsZero())
	base.accounts[0].UpdatedAt = time.Now().Add(time.Millisecond)
	s.applyTicketSchedulingSignal(context.Background(), s.schedulingPending[1])
	a, _ = r.GetByID(context.Background(), 1)
	require.False(t, a.Schedulable)
	require.Equal(t, 1, r.writes)
}

// 故障时保留已知范围：不开启的账号、API Key及其它模型不受新门控影响。
func TestCodexTicketRegressionConfigFailureScopeAndRecovery(t *testing.T) {
	s, r, _ := setupVerifiedTicketTest(t, true)
	a, _ := r.GetByID(context.Background(), 1)
	global := s.config.Load()
	s.markConfigUnavailable()
	require.Same(t, global, s.config.Load())
	require.True(t, s.Blocks(context.Background(), a, "gpt-6-astra"))
	require.False(t, s.Blocks(context.Background(), a, "unconfigured-model"))
	api := *a
	api.Type = AccountTypeAPIKey
	require.False(t, s.Blocks(context.Background(), &api, "gpt-6-astra"))
	copy := *global
	copy.loadedAt = time.Now()
	s.publishConfig(&copy)
	require.NotNil(t, s.enabledAccountConfig(1))
	require.False(t, s.configUnavailable.Load())
	no := false
	_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{Enabled: &no})
	require.NoError(t, err)
	s.markConfigUnavailable()
	require.False(t, s.Blocks(context.Background(), a, "gpt-6-astra"), "已明确关闭时保持原业务")
}

func TestCodexTicketRegressionManualFinishedAccountsResumeAuto(t *testing.T) {
	s, r, _ := setupTicketManualTest(t, 292)
	second := ticketAccount()
	second.ID = 2
	r.accounts = append(r.accounts, second)
	request := manualRequest(s)
	request.AccountIDs = []int64{1, 2}
	m, err := s.PrepareManualCollection(context.Background(), request)
	require.NoError(t, err)
	defer m.Close()
	for _, id := range request.AccountIDs {
		require.True(t, s.manualAccountReserved(context.Background(), id))
	}
	for range 2 {
		ok, err := m.renewSelection(context.Background(), 1)
		require.NoError(t, err)
		require.True(t, ok)
	}
	require.False(t, s.manualAccountReserved(context.Background(), 1))
	require.True(t, s.manualAccountReserved(context.Background(), 2))
	require.False(t, s.manualAccountReserved(context.Background(), 3))
}

// 重试期间被管理员禁用时保留次数，但不能把上轮missing继续冒充本次终态。
func TestCodexTicketRegressionManualLastAttemptAndNewEligibility(t *testing.T) {
	s, r, u := setupTicketManualTest(t, 292)
	zero := 0
	models := []string{"gpt-6-astra"}
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{Models: &models, MaxAttempts: &zero}}})
	require.NoError(t, err)
	u.ticket = "gAAAAA" + strings.Repeat("x", 350)
	m, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
	require.NoError(t, err)
	defer m.Close()
	task := &codexTicketManualTask{id: 1, model: models[0]}
	events := []CodexTicketAttempt{}
	emit := func(e CodexTicketAttempt) { events = append(events, e) }
	m.collectModel(context.Background(), 1, task.model, emit, task)
	r.accounts[0].Status = "inactive"
	m.collectModel(context.Background(), 1, task.model, emit, task)
	require.True(t, task.done)
	require.Len(t, events, 2)
	require.Equal(t, "result", events[1].Kind)
	require.Equal(t, "skipped", events[1].Status)
	require.Equal(t, "account_inactive", events[1].Reason)
	require.True(t, events[1].Diagnostic.PreviousAttempt)
	require.Equal(t, 1, events[1].Attempt)
	require.Equal(t, 356, events[1].Diagnostic.HeaderLength)
}

// 手动和自动同时运行仍只有四个真实采集槽，同号同模型不并发。
func TestCodexTicketRegressionSharedSlotsBoundedAndReleased(t *testing.T) {
	s, _, _ := setupTicketManualTest(t, 292)
	cfg := s.enabledAccountConfig(1)
	releases := []func(){}
	for i := 0; i < 4; i++ {
		release, err := s.acquireTicketCollectionSlot(context.Background(), cfg, 1, fmt.Sprint(i))
		require.NoError(t, err)
		releases = append(releases, release)
		defer release()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := s.acquireTicketCollectionSlot(ctx, cfg, 2, "another-model")
	require.Error(t, err)
	releases[0]()
	release, err := s.acquireTicketCollectionSlot(context.Background(), cfg, 2, "another-model")
	require.NoError(t, err)
	defer release()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel2()
	_, err = s.acquireTicketCollectionSlot(ctx2, cfg, 2, "another-model")
	require.Error(t, err)
}
