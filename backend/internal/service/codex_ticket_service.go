package service

import (
	"bytes"
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

	"github.com/TokenFlux/TokenRouter/internal/pkg/openai"
	"github.com/google/uuid"
)

const codexTicketSettingsKey = "codex_ticket_runtime"

// CodexTicketCache 与账号 extra/调度缓存完全隔离，票据和探测租约均自动过期。
type CodexTicketCache interface {
	Get(context.Context, string) (string, error)
	GetMany(context.Context, []string) (map[string]string, error)
	Set(context.Context, string, string, time.Duration) error
	Claim(context.Context, string, time.Duration) (bool, error)
}

type codexTicketConfig struct {
	Enabled     bool   `json:"enabled"`
	ProxyCipher string `json:"proxy_cipher"`
	Generation  string `json:"generation"`
	loadedAt    time.Time
}

// 管理响应不返回代理密码或原始票据，只暴露配置是否存在。
type CodexTicketSettings struct {
	Enabled         bool `json:"enabled"`
	ProxyConfigured bool `json:"proxy_configured"`
}
type CodexTicketSettingsUpdate struct {
	Enabled         *bool   `json:"enabled"`
	HarvestProxyURL *string `json:"harvest_proxy_url"`
	ClearProxy      bool    `json:"clear_proxy"`
}
type codexTicketValue struct {
	State     string    `json:"state"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CodexTicketService 是默认关闭的合成票据补充器：不修改调度、不覆盖客户端回合状态。
// @project-doc docs/interfaces/openai_upstream.md#codex_ticket_opt_in
type CodexTicketService struct {
	gateway     *OpenAIGatewayService
	settings    SettingRepository
	cache       CodexTicketCache
	cipher      SecretEncryptor
	config      atomic.Pointer[codexTicketConfig]
	updateMu    sync.Mutex
	lifecycleMu sync.Mutex
	cancel      context.CancelFunc
	done        chan struct{}
	stopped     bool
	roundCancel context.CancelFunc
	roundID     string
	roundWG     sync.WaitGroup
	cursor      int64
}

func ProvideCodexTicketService(gateway *OpenAIGatewayService, settings SettingRepository, cache CodexTicketCache, cipher SecretEncryptor) *CodexTicketService {
	s := &CodexTicketService{gateway: gateway, settings: settings, cache: cache, cipher: cipher}
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
	return CodexTicketSettings{Enabled: cfg.Enabled, ProxyConfigured: cfg.ProxyCipher != ""}, nil
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
	if input.ClearProxy {
		cfg.ProxyCipher = ""
	}
	if input.HarvestProxyURL != nil && strings.TrimSpace(*input.HarvestProxyURL) != "" {
		raw := strings.TrimSpace(*input.HarvestProxyURL)
		if err = validateCodexHarvestProxy(raw); err != nil {
			return CodexTicketSettings{}, err
		}
		if s.cipher == nil {
			return CodexTicketSettings{}, errors.New("代理加密服务不可用")
		}
		cfg.ProxyCipher, err = s.cipher.Encrypt(raw)
		if err != nil {
			return CodexTicketSettings{}, errors.New("保存代理凭据失败")
		}
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
	return CodexTicketSettings{Enabled: cfg.Enabled, ProxyConfigured: cfg.ProxyCipher != ""}, nil
}

func (s *CodexTicketService) publishConfig(cfg *codexTicketConfig) {
	old := s.config.Swap(cfg)
	if old == nil || old.Generation != cfg.Generation || !cfg.Enabled {
		s.lifecycleMu.Lock()
		if s.roundCancel != nil {
			s.roundCancel()
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
func validCodexTicket(value codexTicketValue) bool {
	return len(value.State) == 292 && strings.HasPrefix(value.State, "gAAAAA") && !strings.ContainsAny(value.State, "\r\n\x00") && time.Now().Before(value.ExpiresAt)
}

// Apply 只补空状态头：真实会话优先；没有票据或缓存故障不影响原调度与转发。
func (s *CodexTicketService) Apply(ctx context.Context, account *Account, model string, headers http.Header) {
	cfg := s.enabledConfig()
	if cfg == nil || s.cache == nil || s.cipher == nil || headers == nil || !codexTicketAccount(account) || !codexTicketModel(model) || headers.Get(openAICodexTurnStateHeader) != "" {
		return
	}
	for name := range headers {
		if strings.EqualFold(name, openAICodexTurnStateHeader) {
			return
		}
	}
	token := strings.TrimPrefix(headers.Get("Authorization"), "Bearer ")
	if token == "" || token == headers.Get("Authorization") {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	encoded, err := s.cache.Get(readCtx, codexTicketKey(cfg, account, model, token))
	if err != nil || encoded == "" {
		return
	}
	raw, err := s.cipher.Decrypt(encoded)
	if err != nil {
		return
	}
	var value codexTicketValue
	if json.Unmarshal([]byte(raw), &value) != nil || !validCodexTicket(value) {
		return
	}
	latest := s.enabledConfig()
	if latest == nil || latest.Generation != cfg.Generation {
		return
	}
	headers.Set(openAICodexTurnStateHeader, value.State)
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
		var nextRound time.Time
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
			if s.enabledConfig() != nil && time.Now().After(nextRound) {
				s.startRound(ctx)
				nextRound = time.Now().Add(time.Minute)
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
	done := s.done
	s.lifecycleMu.Unlock()
	if done != nil {
		<-done
	}
	// 轮次最长 30 秒且与生命周期共用取消；等待后台释放并发槽和临时连接。
	s.roundWG.Wait()
}

func (s *CodexTicketService) startRound(ctx context.Context) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.roundCancel != nil || s.stopped {
		return
	}
	roundCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	id := uuid.NewString()
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
	// 集群每轮最多一个实例采集；失败保留租约至到期，避免多实例同时探测整个号池。
	claimed, err := s.cache.Claim(ctx, "round:"+cfg.Generation, 45*time.Second)
	if err != nil || !claimed {
		return
	}
	proxy, err := s.cipher.Decrypt(cfg.ProxyCipher)
	if err != nil || validateCodexHarvestProxy(proxy) != nil {
		return
	}
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
				s.probe(ctx, cfg, &item.account, item.model, proxy)
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

func (s *CodexTicketService) probe(ctx context.Context, cfg *codexTicketConfig, account *Account, model, proxy string) {
	current := s.enabledConfig()
	if current == nil || current.Generation != cfg.Generation || ctx.Err() != nil {
		return
	}
	// 重新读取账号，避免已停调或已换凭据的排队任务继续使用老快照。
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
	if encrypted, err := s.cache.Get(ctx, key); err == nil && encrypted != "" {
		if raw, err := s.cipher.Decrypt(encrypted); err == nil {
			var value codexTicketValue
			if json.Unmarshal([]byte(raw), &value) == nil && validCodexTicket(value) && time.Until(value.ExpiresAt) > 10*time.Minute {
				return
			}
		}
	}
	// 账号/模型至少一分钟一次；不在请求热路径同步打票或换代理无限重试。
	claimed, err := s.cache.Claim(ctx, "probe:"+key, time.Minute)
	if err != nil || !claimed {
		return
	}
	probeCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	if s.gateway.concurrencyService != nil {
		slot, err := s.gateway.concurrencyService.AcquireAccountSlot(probeCtx, fresh.ID, fresh.Concurrency)
		if err != nil || slot == nil || !slot.Acquired {
			return
		}
		defer slot.ReleaseFunc()
	}
	body, _ := json.Marshal(map[string]any{"model": model, "store": false, "stream": true, "instructions": "Reply with exactly: pong", "input": []any{map[string]any{"role": "user", "content": "ping"}}})
	probeCtx = WithHTTPUpstreamRedirectsDisabled(WithHTTPUpstreamProfile(probeCtx, HTTPUpstreamProfileOpenAIHarvest))
	req, err := http.NewRequestWithContext(probeCtx, http.MethodPost, chatgptCodexURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Close = true
	req.Host = "chatgpt.com"
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("session_id", uuid.NewString())
	if resolveAndSetOpenAIChatGPTAccountHeaders(probeCtx, s.gateway.accountRepo, req.Header, fresh) != nil {
		return
	}
	ensureCodexIdentityHeaders(req.Header)
	enforceCodexIdentityHeaders(req.Header)
	if model == "gpt-6-astra" && CompareVersions(req.Header.Get("version"), "0.153.4") < 0 {
		// 仅合成探测对齐该模型的上游版本下限，不改正常业务客户端身份。
		req.Header.Set("version", "0.153.4")
		req.Header.Set("user-agent", openai.CodexDefaultOriginator+"/0.153.4 (Ubuntu 22.4.0; x86_64) xterm-256color")
		req.Header.Set("originator", openai.CodexDefaultOriginator)
	}
	// 合成采集使用独立不复用连接，不更改正常业务的 TLS 指纹/代理或身份。
	s.recordObservation(probeCtx, key, "collecting", "", nil)
	state, reason := "failed", "network"
	var expires *time.Time
	defer func() {
		if probeCtx.Err() != nil && state != "ready" {
			reason = "cancelled"
		}
		s.recordObservation(probeCtx, key, state, reason, expires)
	}()
	resp, err := s.gateway.httpUpstream.Do(req, proxy, fresh.ID, fresh.Concurrency)
	if err != nil || resp == nil {
		return
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	value := codexTicketValue{State: extractOpenAICodexTurnState(resp.Header), ExpiresAt: time.Now().Add(time.Hour)}
	if resp.StatusCode != http.StatusOK {
		reason = "upstream"
		return
	}
	if !validCodexTicket(value) {
		state, reason = "missing", "invalid_ticket"
		return
	}
	latest := s.enabledConfig()
	if latest == nil || latest.Generation != cfg.Generation {
		reason = "cancelled"
		return
	}
	// 换凭据/删除账号后的在途探测不进入新账号的票据键。
	reason = "cancelled"
	final, err := s.gateway.accountRepo.GetByID(ctx, account.ID)
	if err != nil || !codexTicketAccount(final) || final.GetOpenAIAccessToken() != token || !final.IsSchedulable() || !final.IsModelSupported(model) {
		return
	}
	if codexTicketKey(cfg, final, model, token) != key {
		return
	}
	raw, _ := json.Marshal(value)
	reason = "storage"
	encrypted, err := s.cipher.Encrypt(string(raw))
	if err != nil {
		return
	}
	if s.cache.Set(ctx, key, encrypted, time.Hour) == nil {
		state, reason, expires = "ready", "", &value.ExpiresAt
	}
}
