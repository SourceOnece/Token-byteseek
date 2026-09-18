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
}

type codexTicketConfig struct {
	Enabled              bool               `json:"enabled"`
	ProxyCipher          string             `json:"proxy_cipher"`
	Generation           string             `json:"generation"`
	Proxies              []codexTicketProxy `json:"proxies,omitempty"`
	SelectionMode        string             `json:"selection_mode,omitempty"`
	FixedProxyID         string             `json:"fixed_proxy_id,omitempty"`
	ProbeIntervalSeconds int                `json:"probe_interval_seconds,omitempty"`
	MaxAttempts          int                `json:"max_attempts,omitempty"`
	TargetLength         int                `json:"target_length,omitempty"`
	RetryIntervalSeconds int                `json:"retry_interval_seconds,omitempty"`
	loadedAt             time.Time
}

// 管理响应不返回代理密码或原始票据，只暴露配置是否存在。
type CodexTicketSettings struct {
	Enabled              bool                   `json:"enabled"`
	ProxyConfigured      bool                   `json:"proxy_configured"`
	Proxies              []CodexTicketProxyView `json:"proxies"`
	SelectionMode        string                 `json:"selection_mode"`
	FixedProxyID         string                 `json:"fixed_proxy_id"`
	ProbeIntervalSeconds int                    `json:"probe_interval_seconds"`
	MaxAttempts          int                    `json:"max_attempts"`
	TargetLength         int                    `json:"target_length"`
	RetryIntervalSeconds int                    `json:"retry_interval_seconds"`
	Revision             string                 `json:"revision"`
}
type CodexTicketSettingsUpdate struct {
	Enabled              *bool                     `json:"enabled"`
	HarvestProxyURL      *string                   `json:"harvest_proxy_url"`
	ClearProxy           bool                      `json:"clear_proxy"`
	Proxies              *[]CodexTicketProxyUpdate `json:"proxies"`
	SelectionMode        *string                   `json:"selection_mode"`
	FixedProxyID         *string                   `json:"fixed_proxy_id"`
	ProbeIntervalSeconds *int                      `json:"probe_interval_seconds"`
	MaxAttempts          *int                      `json:"max_attempts"`
	TargetLength         *int                      `json:"target_length"`
	RetryIntervalSeconds *int                      `json:"retry_interval_seconds"`
	Revision             *string                   `json:"revision"`
}
type codexTicketValue struct {
	State     string    `json:"state"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CodexTicketService 默认关闭；开启后沿快照按账号/模型门控并覆盖回合状态，不改账号总开关。
// @project-doc docs/interfaces/openai_upstream.md#codex_ticket_opt_in
type CodexTicketService struct {
	gateway      *OpenAIGatewayService
	settings     SettingRepository
	cache        CodexTicketCache
	cipher       SecretEncryptor
	config       atomic.Pointer[codexTicketConfig]
	updateMu     sync.Mutex
	lifecycleMu  sync.Mutex
	cancel       context.CancelFunc
	done         chan struct{}
	stopped      bool
	roundCancel  context.CancelFunc
	roundID      string
	roundWG      sync.WaitGroup
	cursor       int64
	nextRound    time.Time
	cacheRetryAt atomic.Int64
	manualCancel context.CancelFunc
	manualID     string
	manualWG     sync.WaitGroup
	ipProber     ProxyExitInfoProber
}

func ProvideCodexTicketService(gateway *OpenAIGatewayService, settings SettingRepository, cache CodexTicketCache, cipher SecretEncryptor, ipProber ProxyExitInfoProber) *CodexTicketService {
	s := &CodexTicketService{gateway: gateway, settings: settings, cache: cache, cipher: cipher, ipProber: ipProber}
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
	if err = s.updateProxySettings(cfg, input); err != nil {
		return CodexTicketSettings{}, err
	}
	if cfg.Enabled && (cfg.ProxyCipher == "" || s.cache == nil || s.cipher == nil) {
		return CodexTicketSettings{}, errors.New("开启前请先配置采集代理及 Redis")
	}
	// 每次保存换代；禁用再启用、换代理都不能复用旧一代票据或在途探测结果。
	cfg.Generation = uuid.NewString()
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return CodexTicketSettings{}, err
	}
	if err = s.settings.Set(ctx, codexTicketSettingsKey, string(encoded)); err != nil {
		return CodexTicketSettings{}, errors.New("保存票据配置失败")
	}
	s.publishConfig(cfg)
	return codexTicketSettingsView(cfg), nil
}

func (s *CodexTicketService) publishConfig(cfg *codexTicketConfig) {
	old := s.config.Swap(cfg)
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
	// 设置轮询失联时宁可不采集/不注入，不能无限沿用曾开启的快照。
	if cfg == nil || !cfg.Enabled || cfg.Generation == "" || time.Since(cfg.loadedAt) > 10*time.Second {
		return nil
	}
	return cfg
}

func codexTicketAccount(account *Account) bool {
	return account != nil && account.ID > 0 && account.IsOpenAIOAuth() && !account.IsCredentialShadow() && !account.IsOpenAIAgentIdentity()
}
func codexTicketModel(model string) bool { return model == "gpt-6-astra" || model == "gpt-5.6-sol" }
func codexTicketKey(cfg *codexTicketConfig, account *Account, model, token string) string {
	// 上游凭据与工作区变动会换键；不在 Redis key 暴露 token 或客户端会话。
	raw, _ := json.Marshal([]any{cfg.Generation, account.ID, model, token, account.GetCredential("chatgpt_account_id"), account.GetCredential("organization_id")})
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

// Apply 沿快照覆盖回合头；缺票不现场采集，由后台续采后恢复。
func (s *CodexTicketService) Apply(ctx context.Context, account *Account, model string, headers http.Header) error {
	cfg := s.enabledConfig()
	if cfg == nil || headers == nil || !codexTicketAccount(account) || !codexTicketModel(model) {
		return nil
	}
	token := strings.TrimPrefix(headers.Get("Authorization"), "Bearer ")
	if token == "" || token == headers.Get("Authorization") {
		return ErrCodexTicketUnavailable
	}
	value, ok := s.lookup(ctx, cfg, account, model, token)
	latest := s.enabledConfig()
	if latest == nil {
		return nil
	}
	if latest.Generation != cfg.Generation || !ok {
		return ErrCodexTicketUnavailable
	}
	// 清理大小写不同的旧键，避免覆盖后实际发出两个状态头。
	for name := range headers {
		if strings.EqualFold(name, openAICodexTurnStateHeader) {
			delete(headers, name)
		}
	}
	headers.Set(openAICodexTurnStateHeader, value.State)
	return nil
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
	encoded, err := s.cache.Get(readCtx, codexTicketKey(cfg, account, model, token))
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
	ok := json.Unmarshal([]byte(raw), &value) == nil && validCodexTicket(value, cfg.targetLength())
	return value, ok
}

// Blocks 只读票据，不刷新凭据、不触发采集，也不更新 schedulable。
func (s *CodexTicketService) Blocks(ctx context.Context, account *Account, model string) bool {
	cfg := s.enabledConfig()
	if cfg == nil || !codexTicketAccount(account) || !codexTicketModel(model) {
		return false
	}
	_, ok := s.lookup(ctx, cfg, account, model, account.GetOpenAIAccessToken())
	latest := s.enabledConfig()
	return latest != nil && (latest.Generation != cfg.Generation || !ok)
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
				cfg = &codexTicketConfig{loadedAt: time.Now()}
			}
			s.publishConfig(cfg)
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
}

func (s *CodexTicketService) startRound(ctx context.Context) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.roundCancel != nil || s.manualCancel != nil || s.stopped || time.Now().Before(s.nextRound) {
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
					s.nextRound = time.Now().Add(latest.interval())
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
	start := sort.Search(len(accounts), func(i int) bool { return accounts[i].ID > s.cursor })
	if start == len(accounts) {
		start = 0
	}
	type job struct {
		account Account
		model   string
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
	for offset := 0; offset < len(accounts); offset++ {
		account := accounts[(start+offset)%len(accounts)]
		if !codexTicketAccount(&account) || !account.IsSchedulable() {
			continue
		}
		s.cursor = account.ID
		for _, model := range []string{"gpt-6-astra", "gpt-5.6-sol"} {
			if !account.IsModelSupported(model) {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case jobs <- job{account: account, model: model}:
			}
		}
	}
}

func (s *CodexTicketService) probe(ctx context.Context, cfg *codexTicketConfig, account *Account, model string) {
	current := s.enabledConfig()
	if current == nil || current.Generation != cfg.Generation || ctx.Err() != nil {
		return
	}
	fresh, err := s.gateway.accountRepo.GetByID(ctx, account.ID)
	if err != nil || !codexTicketAccount(fresh) || !fresh.IsSchedulable() || !fresh.IsModelSupported(model) {
		return
	}
	token, _, err := s.gateway.GetAccessToken(ctx, fresh)
	if err != nil || token == "" {
		s.recordObservation(ctx, codexTicketKey(cfg, fresh, model, fresh.GetOpenAIAccessToken()), "failed", "credential", nil)
		return
	}
	key := codexTicketKey(cfg, fresh, model, token)
	cached, err := s.cache.GetMany(ctx, []string{key, "status:" + key})
	if err != nil {
		return
	}
	if encrypted := cached[key]; encrypted != "" {
		if raw, err := s.cipher.Decrypt(encrypted); err == nil {
			var value codexTicketValue
			if json.Unmarshal([]byte(raw), &value) == nil && validCodexTicket(value, cfg.targetLength()) && time.Until(value.ExpiresAt) > 10*time.Minute {
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
	previousID := ""
	if previous.Diagnostic != nil {
		previousID = previous.Diagnostic.ProxyID
	}
	failed := previous.State == "failed" || previous.State == "missing"
	// 每轮最多三十秒；没票才切换重试，成功即停止，不重放任何用户业务请求。
	cycleCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for attempt := 1; attempt <= cfg.attempts(); attempt++ {
		current = s.enabledConfig()
		if current == nil || current.Generation != cfg.Generation || cycleCtx.Err() != nil {
			return
		}
		proxy, ok := selectCodexTicketProxy(cfg, previousID, failed)
		if !ok {
			return
		}
		ready, retry := s.probeAttempt(cycleCtx, cfg, fresh, model, token, key, proxy, attempt)
		if ready || !retry || attempt == cfg.attempts() {
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
