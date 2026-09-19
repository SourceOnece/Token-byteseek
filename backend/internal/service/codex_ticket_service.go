package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const codexTicketSettingsKey = "codex_ticket_runtime"

// CodexTicketCache 与账号 extra/调度缓存完全隔离，票据和探测租约均自动过期。
type CodexTicketCache interface {
	Get(context.Context, string) (string, error)
	GetMany(context.Context, []string) (map[string]string, error)
	Set(context.Context, string, string, time.Duration) error
	Claim(context.Context, string, time.Duration) (bool, error)
	AcquireLease(context.Context, string, string, time.Duration) (bool, error)
	ReleaseLease(context.Context, string, string) error
	RenewLease(context.Context, string, string, time.Duration) (bool, error)
	LeaseValue(context.Context, string) (string, error)
	ReplaceLease(context.Context, string, string, string, time.Duration) (bool, error)
	SetLatest(context.Context, string, string, int64, time.Duration) error
}

type codexTicketConfig struct {
	ImportDefaults        *codexTicketAccountConfig `json:"import_defaults,omitempty"`
	VerifiedFlow          bool                      `json:"-"`
	FailureThreshold      int                       `json:"failure_threshold,omitempty"`
	CooldownSeconds       int                       `json:"cooldown_seconds,omitempty"`
	CollectionConcurrency int                       `json:"collection_concurrency,omitempty"`
	CacheMinutes          int                       `json:"cache_minutes,omitempty"`
	RefreshBeforeMinutes  *int                      `json:"refresh_before_minutes,omitempty"`
	ProxyPolicy           *codexTicketProxyPolicy   `json:"proxy_policy,omitempty"`
	unlimitedAttempts     bool
	stored                *string
	Accounts              map[string]codexTicketAccountConfig `json:"accounts,omitempty"`
	WatchdogMode          string                              `json:"watchdog_mode,omitempty"`
	accountID             int64
	Enabled               bool               `json:"enabled"`
	ProxyCipher           string             `json:"proxy_cipher"`
	Generation            string             `json:"generation"`
	Proxies               []codexTicketProxy `json:"proxies,omitempty"`
	SelectionMode         string             `json:"selection_mode,omitempty"`
	FixedProxyID          string             `json:"fixed_proxy_id,omitempty"`
	ProbeIntervalSeconds  int                `json:"probe_interval_seconds,omitempty"`
	MaxAttempts           int                `json:"max_attempts,omitempty"`
	TargetLength          int                `json:"target_length,omitempty"`
	DegradedSignalLength  int                `json:"degraded_signal_length,omitempty"`
	Models                []string           `json:"models,omitempty"`
	RetryIntervalSeconds  int                `json:"retry_interval_seconds,omitempty"`
	loadedAt              time.Time
}

// 管理响应不返回代理密码或原始票据，只暴露配置是否存在。
type CodexTicketSettings struct {
	AccountRules           []CodexTicketAccountSettings `json:"account_rules,omitempty"`
	ProxyPolicy            CodexTicketProxyPolicyView   `json:"proxy_policy"`
	AccountProxyConfigured bool                         `json:"account_proxy_configured"`
	WatchdogMode           string                       `json:"watchdog_mode"`
	Enabled                bool                         `json:"enabled"`
	ProxyConfigured        bool                         `json:"proxy_configured"`
	Proxies                []CodexTicketProxyView       `json:"proxies"`
	SelectionMode          string                       `json:"selection_mode"`
	FixedProxyID           string                       `json:"fixed_proxy_id"`
	ProbeIntervalSeconds   int                          `json:"probe_interval_seconds"`
	MaxAttempts            int                          `json:"max_attempts"`
	TargetLength           int                          `json:"target_length"`
	DegradedSignalLength   int                          `json:"degraded_signal_length"`
	Models                 []string                     `json:"models"`
	RetryIntervalSeconds   int                          `json:"retry_interval_seconds"`
	Revision               string                       `json:"revision"`
}
type CodexTicketSettingsUpdate struct {
	ProxyPolicy          *CodexTicketProxyPolicyUpdate `json:"proxy_policy"`
	WatchdogMode         *string                       `json:"watchdog_mode"`
	Enabled              *bool                         `json:"enabled"`
	HarvestProxyURL      *string                       `json:"harvest_proxy_url"`
	ClearProxy           bool                          `json:"clear_proxy"`
	Proxies              *[]CodexTicketProxyUpdate     `json:"proxies"`
	SelectionMode        *string                       `json:"selection_mode"`
	FixedProxyID         *string                       `json:"fixed_proxy_id"`
	ProbeIntervalSeconds *int                          `json:"probe_interval_seconds"`
	MaxAttempts          *int                          `json:"max_attempts"`
	TargetLength         *int                          `json:"target_length"`
	DegradedSignalLength *int                          `json:"degraded_signal_length"`
	Models               *[]string                     `json:"models"`
	RetryIntervalSeconds *int                          `json:"retry_interval_seconds"`
	Revision             *string                       `json:"revision"`
}
type codexTicketValue struct {
	Verified            bool   `json:"verified,omitempty"`
	BusinessFingerprint string `json:"business_fingerprint,omitempty"`
	Attempts            int    `json:"attempts,omitempty"`
	encoded             string
	State               string    `json:"state"`
	ExpiresAt           time.Time `json:"expires_at"`
}

// CodexTicketService 默认关闭；按账号/模型门控并覆盖回合状态，明确长度结论按独立联动更新总调度。
// @project-doc docs/interfaces/openai_upstream.md#codex_ticket_opt_in
type CodexTicketService struct {
	schedulingMu        sync.Mutex
	schedulingPending   map[int64]ticketSchedulingSignal
	schedulingWake      chan struct{}
	schedulingWG        sync.WaitGroup
	schedulingDropped   atomic.Uint64
	proxyProviderClient *http.Client
	proxyRepo           ProxyRepository
	gateway             *OpenAIGatewayService
	settings            SettingRepository
	cache               CodexTicketCache
	cipher              SecretEncryptor
	config              atomic.Pointer[codexTicketConfig]
	configUnavailable   atomic.Bool
	updateMu            sync.Mutex
	lifecycleMu         sync.Mutex
	cancel              context.CancelFunc
	done                chan struct{}
	stopped             bool
	roundCancel         context.CancelFunc
	roundID             string
	roundWG             sync.WaitGroup
	cursor              int64
	cursorModel         string
	nextRound           time.Time
	cacheRetryAt        atomic.Int64
	watchdogWake        atomic.Bool
	manualCancel        context.CancelFunc
	manualID            string
	manualWG            sync.WaitGroup
	ipProber            ProxyExitInfoProber
}

func ProvideCodexTicketService(gateway *OpenAIGatewayService, settings SettingRepository, cache CodexTicketCache, cipher SecretEncryptor, ipProber ProxyExitInfoProber, proxies ProxyRepository) *CodexTicketService {
	s := &CodexTicketService{gateway: gateway, settings: settings, cache: cache, cipher: cipher, ipProber: ipProber, proxyRepo: proxies}
	gateway.codexTickets.Store(s)
	s.Start()
	return s
}

func validateCodexHarvestProxy(raw string) error {
	if len(raw) > 4096 {
		return errors.New("采集代理地址过长")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return errors.New("采集代理须为完整 HTTP(S)/SOCKS5(h) 地址，不能含路径、查询或片段")
	}
	switch parsed.Scheme {
	case "http", "https", "socks5", "socks5h":
	default:
		return errors.New("不支持的采集代理协议")
	}
	if p := parsed.Port(); p != "" {
		n, e := strconv.Atoi(p)
		if e != nil || n < 1 || n > 65535 {
			return errors.New("无效的采集代理端口")
		}
	}
	return nil
}

func (s *CodexTicketService) readConfig(ctx context.Context) (*codexTicketConfig, error) {
	raw, err := s.settings.GetValue(ctx, codexTicketSettingsKey)
	if errors.Is(err, ErrSettingNotFound) {
		return &codexTicketConfig{loadedAt: time.Now()}, nil
	}
	if err != nil {
		return nil, errors.New("读取票据配置失败")
	}
	cfg := &codexTicketConfig{}
	if json.Unmarshal([]byte(raw), cfg) != nil {
		return nil, errors.New("票据配置格式无效")
	}
	cfg.loadedAt = time.Now()
	cfg.stored = &raw
	return cfg, nil
}

func (s *CodexTicketService) View(ctx context.Context) (CodexTicketSettings, error) {
	cfg, err := s.readConfig(ctx)
	if err != nil {
		return CodexTicketSettings{}, err
	}
	return codexTicketSettingsView(cfg), nil
}

func (s *CodexTicketService) Update(ctx context.Context, input CodexTicketSettingsUpdate) (CodexTicketSettings, error) {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	cfg, err := s.readConfig(ctx)
	if err != nil {
		return CodexTicketSettings{}, err
	}
	if input.Enabled != nil {
		cfg.Enabled = *input.Enabled
	}
	if input.WatchdogMode != nil {
		if !validTicketWatchdogMode(*input.WatchdogMode, false) {
			return CodexTicketSettings{}, errors.New("无效的守护模式")
		}
		cfg.WatchdogMode = *input.WatchdogMode
	}
	if input.ProxyPolicy == nil {
		if err = s.updateProxySettings(cfg, input); err != nil {
			return CodexTicketSettings{}, err
		}
	} else if input.Revision != nil && *input.Revision != cfg.Generation {
		return CodexTicketSettings{}, errors.New("票据配置已更新，请重新加载")
	}
	if input.ProxyPolicy != nil {
		policy, e := s.updateTicketProxyPolicy(cfg, input.ProxyPolicy)
		if e != nil {
			return CodexTicketSettings{}, e
		}
		cfg.applyProxyPolicy(policy)
	}
	if cfg.Enabled && (!cfg.hasCollectionProxy() || s.cache == nil || s.cipher == nil) {
		return CodexTicketSettings{}, errors.New("开启前请先配置采集代理及 Redis")
	}
	// 每次保存换代；禁用再启用、换代理都不能复用旧一代票据或在途探测结果。
	cfg.Generation = uuid.NewString()
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return CodexTicketSettings{}, err
	}
	if err = s.persistTicketConfig(ctx, cfg, string(encoded)); err != nil {
		return CodexTicketSettings{}, err
	}
	s.publishConfig(cfg)
	return codexTicketSettingsView(cfg), nil
}

func (s *CodexTicketService) publishConfig(cfg *codexTicketConfig) {
	old := s.config.Swap(cfg)
	s.configUnavailable.Store(false)
	if old == nil || old.Generation != cfg.Generation || !cfg.Enabled {
		s.lifecycleMu.Lock()
		s.nextRound = time.Time{}
		if s.roundCancel != nil {
			s.roundCancel()
		}
		if s.manualCancel != nil {
			s.manualCancel()
		}
		s.lifecycleMu.Unlock()
	}
}
func (s *CodexTicketService) enabledConfig() *codexTicketConfig {
	if s == nil {
		return nil
	}
	cfg := s.config.Load()
	// 设置失联停止维护；业务另走routingConfig，不能把失联解释为管理员关闭。
	if cfg == nil || !cfg.Enabled || cfg.Generation == "" || s.configUnavailable.Load() || time.Since(cfg.loadedAt) > 10*time.Second {
		return nil
	}
	return cfg
}

// 保留最后确认的作用域，但配置失联时拒绝注入或绕过票据；已确认关闭和范围外模型保持旧流程。
func (s *CodexTicketService) routingConfig(accountID int64) (*codexTicketConfig, bool) {
	if s == nil {
		return nil, false
	}
	global := s.config.Load()
	if global == nil {
		return nil, true
	}
	cfg := ticketConfigForAccount(global, accountID)
	if !cfg.Enabled {
		return nil, false
	}
	return cfg, s.configUnavailable.Load() || cfg.Generation == "" || time.Since(global.loadedAt) > 10*time.Second
}

// 不发布伪造的关闭配置；保留上次已确认配置供业务门控，并取消在途维护。
func (s *CodexTicketService) markConfigUnavailable() {
	s.configUnavailable.Store(true)
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.roundCancel != nil {
		s.roundCancel()
	}
	if s.manualCancel != nil {
		s.manualCancel()
	}
}

func codexTicketAccount(account *Account) bool {
	return account != nil && account.ID > 0 && account.IsOpenAIOAuth() && !account.IsCredentialShadow() && !account.IsOpenAIAgentIdentity()
}
func codexTicketKey(cfg *codexTicketConfig, account *Account, model, token string) string {
	// 上游凭据与工作区变动会换键；不在 Redis key 暴露 token 或客户端会话。
	parts := []any{cfg.Generation, account.ID, model, token, account.GetCredential("chatgpt_account_id"), account.GetCredential("organization_id")}
	// 新模式单独绑定业务出口；旧模式的键完全不变，不让未验证票混入双链路。
	if cfg.VerifiedFlow {
		parts = append(parts, "verified-flow-v1", ticketBusinessFingerprint(account))
	}
	raw, _ := json.Marshal(parts)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func validCodexTicket(value codexTicketValue, lengths ...int) bool {
	target := 292
	if len(lengths) > 0 {
		target = lengths[0]
	}
	return len(value.State) == target && strings.HasPrefix(value.State, "gAAAAA") && !strings.ContainsAny(value.State, "\r\n\x00") && time.Now().Before(value.ExpiresAt)
}

// 缺票错误只用于当前模型的准入，不写账号状态或质量检测结果。
var ErrCodexTicketUnavailable = errors.New("codex turn-state ticket unavailable")

const CodexTicketUnavailableReason GatewayFailureReason = "codex_ticket_unavailable"

// 选后票据失效属于本地资格变化：允许未输出请求在原预算内换号，不惩罚上游健康。
func newCodexTicketUnavailableError() error {
	return errors.Join(ErrCodexTicketUnavailable, &UpstreamFailoverError{
		StatusCode:        http.StatusServiceUnavailable,
		Stage:             GatewayFailureStageAccountSelection,
		Scope:             GatewayFailureScopeAccount,
		Reason:            CodexTicketUnavailableReason,
		NextAccountAction: NextAccountRetry,
	})
}

// Apply 沿快照覆盖回合头；缺票不现场采集，由后台续采后恢复。
func (s *CodexTicketService) Apply(ctx context.Context, account *Account, model string, headers http.Header) error {
	_, err := s.applyWithReceipt(ctx, account, model, headers)
	return err
}

// 收据绑定真正写入请求的票据密文，不用客户端传入的头作为守护凭据。
func (s *CodexTicketService) applyWithReceipt(ctx context.Context, account *Account, model string, headers http.Header) (*codexTicketReceipt, error) {
	if !codexTicketAccount(account) || headers == nil {
		return nil, nil
	}
	cfg, unavailable := s.routingConfig(account.ID)
	if cfg != nil && !cfg.hasModel(model) {
		return nil, nil
	}
	if unavailable {
		return nil, newCodexTicketUnavailableError()
	}
	if cfg == nil {
		return nil, nil
	}
	token := strings.TrimPrefix(headers.Get("Authorization"), "Bearer ")
	if token == "" || token == headers.Get("Authorization") {
		return nil, newCodexTicketUnavailableError()
	}
	// 新模式在注入前复核当前账号/业务出口，长会话或陈旧快照不能沿旧代理继续用票。
	if cfg.VerifiedFlow {
		if s.gateway == nil || s.gateway.accountRepo == nil {
			return nil, newCodexTicketUnavailableError()
		}
		read, stop := context.WithTimeout(ctx, 200*time.Millisecond)
		live, err := s.gateway.accountRepo.GetByID(read, account.ID)
		stop()
		if err != nil || !codexTicketAccount(live) || !live.IsSchedulable() || !live.IsModelSupported(model) || live.GetOpenAIAccessToken() != token || ticketBusinessFingerprint(live) == "" || codexTicketKey(cfg, live, model, token) != codexTicketKey(cfg, account, model, token) {
			return nil, newCodexTicketUnavailableError()
		}
	}
	value, ok := s.lookup(ctx, cfg, account, model, token)
	latest, unavailable := s.routingConfig(account.ID)
	if unavailable {
		return nil, newCodexTicketUnavailableError()
	}
	if latest == nil {
		return nil, nil
	}
	if latest.Generation != cfg.Generation || !ok {
		return nil, newCodexTicketUnavailableError()
	}
	// 清理大小写不同的旧键，避免覆盖后实际发出两个状态头。
	for name := range headers {
		if strings.EqualFold(name, openAICodexTurnStateHeader) {
			delete(headers, name)
		}
	}
	headers.Set(openAICodexTurnStateHeader, value.State)
	if ticketWatchdogMode(cfg.WatchdogMode) == "off" {
		return nil, nil
	}
	// 长连接只保留本次资格版本，不持有整份号池配置/代理映射。
	receiptConfig := &codexTicketConfig{Generation: cfg.Generation, accountID: account.ID, VerifiedFlow: cfg.VerifiedFlow, WatchdogMode: cfg.WatchdogMode, TargetLength: cfg.targetLength(), DegradedSignalLength: cfg.DegradedSignalLength}
	return &codexTicketReceipt{s: s, cfg: receiptConfig, key: codexTicketKey(cfg, account, model, token), encoded: value.encoded, model: model, stateHash: sha256.Sum256([]byte(value.State))}, nil
}

func (s *CodexTicketService) lookup(ctx context.Context, cfg *codexTicketConfig, account *Account, model, token string) (codexTicketValue, bool) {
	var value codexTicketValue
	if s.cache == nil || s.cipher == nil || token == "" {
		return value, false
	}
	// 缓存失联短暂退避，避免大号池逐号等待完整超时；缺票仍按开启时的门控处理。
	if time.Now().UnixNano() < s.cacheRetryAt.Load() {
		return value, false
	}
	readCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	key := codexTicketKey(cfg, account, model, token)
	encoded, err := s.cache.Get(readCtx, key)
	// 双链路续期冷却只暂停维护；仍有效且未被守护废弃的旧验证票继续供业务使用。
	if err == nil && cfg.FailureThreshold > 0 && !cfg.VerifiedFlow {
		raw, e := s.cache.Get(readCtx, "collection:"+key)
		err = e
		if err == nil && ticketCollectionState(raw).CooldownUntil != nil {
			return value, false
		}
	}
	if err != nil && ctx.Err() == nil {
		s.cacheRetryAt.Store(time.Now().Add(time.Second).UnixNano())
	}
	if err != nil || encoded == "" {
		return value, false
	}
	raw, err := s.cipher.Decrypt(encoded)
	if err != nil {
		return value, false
	}
	ok := json.Unmarshal([]byte(raw), &value) == nil && validTicketForAccount(value, cfg, account)
	value.encoded = encoded
	return value, ok
}

// Blocks 只读票据，不刷新凭据、不触发采集，也不更新 schedulable。
func (s *CodexTicketService) Blocks(ctx context.Context, account *Account, model string) bool {
	if !codexTicketAccount(account) {
		return false
	}
	cfg, unavailable := s.routingConfig(account.ID)
	if cfg != nil && !cfg.hasModel(model) {
		return false
	}
	if unavailable {
		return true
	}
	if cfg == nil {
		return false
	}
	// 调度初筛使用不含令牌/工作区的 sched:meta；不能把摘要缺字段误判为账号缺票。
	// 只补取同 ID 的完整账号，不回填共享摘要，也不降低原票据的身份隔离要求。
	lookupAccount := account
	if account.GetOpenAIAccessToken() == "" || (cfg.VerifiedFlow && account.ProxyID != nil && account.Proxy == nil) {
		lookupAccount = s.resolveTicketLookupAccount(ctx, account.ID)
	}
	// 旧摘要可能缺auth_mode；同ID完整账号确认是特殊授权后，恢复其原有票据豁免。
	if lookupAccount != nil && lookupAccount.ID == account.ID && lookupAccount.IsOpenAIAgentIdentity() {
		return false
	}
	ok := false
	if codexTicketAccount(lookupAccount) && lookupAccount.ID == account.ID {
		_, ok = s.lookup(ctx, cfg, lookupAccount, model, lookupAccount.GetOpenAIAccessToken())
	}
	latest, unavailable := s.routingConfig(account.ID)
	return unavailable || latest != nil && (latest.Generation != cfg.Generation || !ok)
}

// 完整账号优先沿用调度缓存，缺失时走其已有受限数据库回退；不调用带状态更新的资格方法。
func (s *CodexTicketService) resolveTicketLookupAccount(ctx context.Context, accountID int64) *Account {
	if s.gateway == nil {
		return nil
	}
	readCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	var account *Account
	var err error
	if s.gateway.schedulerSnapshot != nil {
		account, err = s.gateway.schedulerSnapshot.GetAccount(readCtx, accountID)
	} else if s.gateway.accountRepo != nil {
		account, err = s.gateway.accountRepo.GetByID(readCtx, accountID)
	}
	if err != nil || readCtx.Err() != nil {
		return nil
	}
	return account
}

func (s *CodexTicketService) Start() {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.done != nil || s.stopped {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	s.schedulingWake = make(chan struct{}, 1)
	s.schedulingWG.Add(1)
	go func() { defer s.schedulingWG.Done(); s.runTicketScheduling(ctx) }()
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			readCtx, stop := context.WithTimeout(ctx, time.Second)
			s.updateMu.Lock()
			cfg, err := s.readConfig(readCtx)
			stop()
			if err != nil {
				s.markConfigUnavailable()
			} else {
				s.publishConfig(cfg)
			}
			s.updateMu.Unlock()
			if s.enabledConfig() != nil {
				s.startRound(ctx)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
func (s *CodexTicketService) Stop() {
	if s == nil {
		return
	}
	s.lifecycleMu.Lock()
	s.stopped = true
	if s.cancel != nil {
		s.cancel()
	}
	if s.roundCancel != nil {
		s.roundCancel()
	}
	if s.manualCancel != nil {
		s.manualCancel()
	}
	done := s.done
	s.lifecycleMu.Unlock()
	if done != nil {
		<-done
	}
	// 轮次最长 30 秒且与生命周期共用取消；等待后台释放并发槽和临时连接。
	s.roundWG.Wait()
	s.manualWG.Wait()
	s.schedulingWG.Wait()
}

func (s *CodexTicketService) startRound(ctx context.Context) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.roundCancel != nil || s.stopped {
		return
	}
	// 手动只保留所选账号，其他账号的自动续期和守护唤醒不再等待整批结束。
	if s.watchdogWake.Swap(false) {
		s.nextRound = time.Time{}
	}
	if time.Now().Before(s.nextRound) {
		return
	}
	roundCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	id := uuid.NewString()
	cfg := s.enabledConfig()
	s.roundCancel = cancel
	s.roundID = id
	s.roundWG.Add(1)
	go func() {
		defer s.roundWG.Done()
		defer cancel()
		defer func() {
			s.lifecycleMu.Lock()
			if s.roundID == id {
				s.roundCancel = nil
				latest := s.enabledConfig()
				if latest != nil && cfg != nil && latest.Generation == cfg.Generation {
					s.nextRound = time.Now().Add(latest.scanInterval())
				}
			}
			s.lifecycleMu.Unlock()
		}()
		s.harvest(roundCtx)
	}()
}

func (s *CodexTicketService) harvest(ctx context.Context) {
	cfg := s.enabledConfig()
	if cfg == nil || s.gateway == nil || s.gateway.accountRepo == nil || s.gateway.httpUpstream == nil || s.cache == nil || s.cipher == nil {
		return
	}
	// 集群每轮最多一个实例采集；正常结束按 owner 解锁，崩溃则由 TTL 释放。
	leaseKey, owner := "round:"+cfg.Generation, uuid.NewString()
	claimed, err := s.cache.AcquireLease(ctx, leaseKey, owner, 45*time.Second)
	if err != nil || !claimed {
		return
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		_ = s.cache.ReleaseLease(releaseCtx, leaseKey, owner)
	}()
	accounts, err := s.gateway.accountRepo.ListByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })
	type job struct {
		account Account
		model   string
	}
	// 不再用全局模型矩阵，按每个账号的完整规则建立公平轮转任务。
	tasks := []job{}
	for _, account := range accounts {
		if !codexTicketCollectionAllowed(ctx, &account) {
			continue
		}
		ac := ticketConfigForAccount(cfg, account.ID)
		if !ac.Enabled {
			continue
		}
		for _, model := range ac.models() {
			if account.IsModelSupported(model) {
				tasks = append(tasks, job{account, model})
			}
		}
	}
	if len(tasks) == 0 {
		return
	}
	start := 0
	for i, item := range tasks {
		if item.account.ID == s.cursor && item.model == s.cursorModel {
			start = (i + 1) % len(tasks)
			break
		}
	}
	jobs := make(chan job)
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for item := range jobs {
				s.probe(ctx, cfg, &item.account, item.model)
			}
		}()
	}
	defer func() { close(jobs); workers.Wait() }()
	for offset := 0; offset < len(tasks); offset++ {
		if ctx.Err() != nil {
			return
		}
		item := tasks[(start+offset)%len(tasks)]
		account, model := item.account, item.model
		if !codexTicketCollectionAllowed(ctx, &account) {
			continue
		}
		if !account.IsModelSupported(model) {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case jobs <- job{account: account, model: model}:
			s.cursor, s.cursorModel = account.ID, model
		}
	}
}

func (s *CodexTicketService) probe(ctx context.Context, cfg *codexTicketConfig, account *Account, model string) {
	if s.manualAccountReserved(ctx, account.ID) {
		return
	}
	cfg = ticketConfigForAccount(cfg, account.ID)
	if !cfg.Enabled || !s.ticketConfigCurrent(cfg) || ctx.Err() != nil {
		return
	}
	fresh, err := s.gateway.accountRepo.GetByID(ctx, account.ID)
	if err != nil || !codexTicketCollectionAllowed(ctx, fresh) || !fresh.IsModelSupported(model) {
		return
	}
	token, _, err := s.gateway.GetAccessToken(ctx, fresh)
	if err != nil || token == "" {
		key := codexTicketKey(cfg, fresh, model, fresh.GetOpenAIAccessToken())
		s.recordObservation(ctx, key, "failed", "credential", nil)
		s.recordLatest(ctx, cfg, key, "auto", CodexTicketAttempt{Status: "failed", Reason: "credential", FinishedAt: time.Now().UTC()})
		return
	}
	key := codexTicketKey(cfg, fresh, model, token)
	cached, err := s.cache.GetMany(ctx, []string{key, "status:" + key, "collection:" + key})
	if err != nil {
		return
	}
	if ticketCollectionState(cached["collection:"+key]).CooldownUntil != nil {
		return
	}
	if encrypted := cached[key]; encrypted != "" {
		if raw, err := s.cipher.Decrypt(encrypted); err == nil {
			var value codexTicketValue
			if json.Unmarshal([]byte(raw), &value) == nil && validTicketForAccount(value, cfg, fresh) && time.Until(value.ExpiresAt) > cfg.refreshBefore() {
				return
			}
		}
	}
	var previous codexTicketObservation
	_ = json.Unmarshal([]byte(cached["status:"+key]), &previous)
	if previous.Diagnostic != nil && previous.Diagnostic.RetryNotBefore != nil && time.Now().Before(*previous.Diagnostic.RetryNotBefore) {
		return
	}
	claimed, err := s.cache.Claim(ctx, "probe:"+key, cfg.interval())
	if err != nil || !claimed {
		return
	}
	// 正上限仍约束当前自动轮次，保留历史周期重试；0的计数跨短轮次延续直到成功。
	if cfg.attempts() > 0 {
		s.resetTicketAttempts(ctx, key)
	}
	previousID := ""
	if previous.Diagnostic != nil {
		previousID = previous.Diagnostic.ProxyID
	}
	failed := previous.State == "failed" || previous.State == "missing"
	// 每轮最多三十秒；没票才切换重试，成功即停止，不重放任何用户业务请求。
	cycleCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	limit := cfg.attempts()
	if limit == 0 {
		limit = 1
	} // 无限采集每轮让出worker，下轮从累计次数继续，防止饿死其它账号。
	for localAttempt := 1; localAttempt <= limit; localAttempt++ {
		if !s.ticketConfigCurrent(cfg) || cycleCtx.Err() != nil || s.manualAccountReserved(cycleCtx, account.ID) {
			return
		}
		proxy, ok := selectCodexTicketProxy(cfg, previousID, failed)
		if !ok {
			return
		}
		ready, retry := s.probeAttempt(cycleCtx, cfg, fresh, model, token, key, proxy, localAttempt)
		if ready {
			s.resetTicketAttempts(cycleCtx, key)
		}
		if ready || !retry || localAttempt == limit {
			return
		}
		previousID, failed = proxy.ID, true
		timer := time.NewTimer(cfg.retryInterval())
		select {
		case <-cycleCtx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
