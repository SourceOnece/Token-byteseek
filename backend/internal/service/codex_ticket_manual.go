package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// 历史仅保存脱敏结果，不保存票据、令牌、代理 URL 或原始错误正文。
type CodexTicketAttempt struct {
	ID           int64                  `json:"id,omitempty"`
	Kind         string                 `json:"kind"`
	AccountID    int64                  `json:"account_id"`
	AccountName  string                 `json:"account_name"`
	Email        string                 `json:"email"`
	Model        string                 `json:"model"`
	TargetLength int                    `json:"target_length"`
	Status       string                 `json:"status"`
	Reason       string                 `json:"reason,omitempty"`
	Attempt      int                    `json:"attempt"`
	StartedAt    time.Time              `json:"started_at"`
	FinishedAt   time.Time              `json:"finished_at"`
	DurationMS   int64                  `json:"duration_ms"`
	Diagnostic   *CodexTicketDiagnostic `json:"diagnostic,omitempty"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
	ReferenceIP  string                 `json:"reference_ip,omitempty"`
	IPCheckedAt  *time.Time             `json:"ip_checked_at,omitempty"`
	IPStatus     string                 `json:"ip_status,omitempty"`
	IPSource     string                 `json:"ip_source,omitempty"`
	IPHTTPStatus int                    `json:"ip_http_status,omitempty"`
}
type CodexTicketManualRun struct {
	ID         string              `json:"id"`
	Status     string              `json:"status"`
	Config     CodexTicketSettings `json:"config"`
	Total      int                 `json:"total"`
	StartedAt  time.Time           `json:"started_at"`
	FinishedAt *time.Time          `json:"finished_at,omitempty"`
	Counts     map[string]int      `json:"counts"`
}
type CodexTicketManualRequest struct {
	AccountIDs []int64 `json:"account_ids"`
	Confirmed  bool    `json:"confirmed"`
	Revision   string  `json:"revision"`
}
type CodexTicketHistoryRepository interface {
	CreateTicketRun(context.Context, *CodexTicketManualRun) error
	AppendTicketEvent(context.Context, string, *CodexTicketAttempt) error
	HeartbeatTicketRun(context.Context, string) error
	FinishTicketRun(context.Context, string, string, map[string]int) error
	ListTicketRuns(context.Context, int) ([]CodexTicketManualRun, error)
	GetTicketRun(context.Context, string) (*CodexTicketManualRun, error)
	ListTicketEvents(context.Context, string, string, string, int64, string, int) ([]CodexTicketAttempt, int, error)
}

// 手动只忽略账号调度开关，不改账号对象、不绕过禁用/过期/冷却等其他保护。
type codexTicketManualContextKey struct{}

func codexTicketCollectionAllowed(ctx context.Context, account *Account) bool {
	if !codexTicketAccount(account) {
		return false
	}
	if manual, _ := ctx.Value(codexTicketManualContextKey{}).(bool); manual {
		copy := *account
		copy.Schedulable = true
		return copy.IsSchedulable()
	}
	return account.IsSchedulable()
}

func (r *CodexTicketManualRequest) Normalize() error {
	if !r.Confirmed {
		return errors.New("请先确认采集将消耗上游额度")
	}
	if len(r.AccountIDs) < 1 || len(r.AccountIDs) > 500 {
		return errors.New("每批选择 1–500 个账号")
	}
	seen := map[int64]bool{}
	ids := make([]int64, 0, len(r.AccountIDs))
	for _, id := range r.AccountIDs {
		if id <= 0 {
			return errors.New("账号 ID 无效")
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	r.AccountIDs = ids
	return nil
}

func (s *CodexTicketService) TicketHistory() (CodexTicketHistoryRepository, error) {
	if s == nil || s.gateway == nil {
		return nil, errors.New("票据服务不可用")
	}
	r, ok := s.gateway.accountRepo.(CodexTicketHistoryRepository)
	if !ok {
		return nil, errors.New("采集历史服务不可用")
	}
	return r, nil
}

type CodexTicketManualSession struct {
	s         *CodexTicketService
	cfg       *codexTicketConfig
	repo      CodexTicketHistoryRepository
	ctx       context.Context
	cancel    context.CancelFunc
	ids       []int64
	Run       CodexTicketManualRun
	locked    bool
	closeOnce sync.Once
}

// @project-doc docs/interfaces/codex_ticket.md#manual_collection
// 手动入口不绕过总开关；冻结账号与配置，保存设置/停机/断连后停止余下尝试。
func (s *CodexTicketService) PrepareManualCollection(ctx context.Context, req CodexTicketManualRequest) (*CodexTicketManualSession, error) {
	if err := req.Normalize(); err != nil {
		return nil, err
	}
	repo, err := s.TicketHistory()
	if err != nil {
		return nil, err
	}
	s.updateMu.Lock()
	cfg, err := s.readConfig(ctx)
	if err == nil {
		s.publishConfig(cfg)
	}
	s.updateMu.Unlock()
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled || len(cfg.proxies()) == 0 {
		return nil, errors.New("请先在网关服务 OpenAI 中开启票据采集并配置代理")
	}
	if req.Revision == "" || req.Revision != cfg.Generation {
		return nil, errors.New("采集配置已变化，请重新打开弹窗确认")
	}
	if s.cache == nil || s.cipher == nil || s.gateway.httpUpstream == nil {
		return nil, errors.New("采集服务不可用")
	}
	ctx, cancel := context.WithCancel(ctx)
	m := &CodexTicketManualSession{s: s, cfg: cfg, repo: repo, ctx: ctx, cancel: cancel, ids: append([]int64(nil), req.AccountIDs...), Run: CodexTicketManualRun{ID: uuid.NewString(), Status: "running", Config: codexTicketSettingsView(cfg), Total: len(req.AccountIDs) * len(cfg.models()), StartedAt: time.Now().UTC(), Counts: map[string]int{}}}
	s.lifecycleMu.Lock()
	if s.stopped || s.manualCancel != nil {
		s.lifecycleMu.Unlock()
		cancel()
		return nil, errors.New("已有手动采集正在执行，请稍后重试")
	}
	s.manualCancel = cancel
	s.manualID = m.Run.ID
	s.manualWG.Add(1)
	s.lifecycleMu.Unlock()
	// 等待已在途自动轮次自然结束，不打断现有采集；等待本身不发上游请求。
	waitCtx, stop := context.WithTimeout(ctx, 35*time.Second)
	defer stop()
	for {
		ok, e := s.cache.AcquireLease(waitCtx, "round:"+cfg.Generation, m.Run.ID, 45*time.Second)
		if e != nil {
			m.Close()
			return nil, errors.New("采集锁不可用")
		}
		if ok {
			m.locked = true
			break
		}
		select {
		case <-waitCtx.Done():
			m.Close()
			return nil, errors.New("后台采集繁忙，请稍后重试")
		case <-time.After(500 * time.Millisecond):
		}
	}
	if latest := s.enabledConfig(); latest == nil || latest.Generation != cfg.Generation || ctx.Err() != nil {
		m.Close()
		return nil, errors.New("采集配置已变化或请求已取消")
	}
	if err = repo.CreateTicketRun(ctx, &m.Run); err != nil {
		m.Close()
		return nil, errors.New("创建采集历史失败，未发起采集")
	}
	return m, nil
}
func (m *CodexTicketManualSession) Close() {
	m.closeOnce.Do(func() {
		m.cancel()
		if m.locked {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = m.s.cache.ReleaseLease(ctx, "round:"+m.cfg.Generation, m.Run.ID)
			cancel()
		}
		m.s.lifecycleMu.Lock()
		if m.s.manualID == m.Run.ID {
			m.s.manualCancel = nil
			m.s.manualID = ""
		}
		m.s.lifecycleMu.Unlock()
		m.s.manualWG.Done()
	})
}

// 单条日志先落库再发 SSE；断连后仍保存已完成结果，不把未执行任务伪装成失败。
func (m *CodexTicketManualSession) Execute(emit func(string, any) bool) {
	defer m.Close()
	ctx, cancel := context.WithCancel(m.ctx)
	defer cancel()
	renewDone := make(chan struct{})
	go func() {
		defer close(renewDone)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				leaseCtx, stop := context.WithTimeout(ctx, 3*time.Second)
				ok, err := m.s.cache.RenewLease(leaseCtx, "round:"+m.cfg.Generation, m.Run.ID, 45*time.Second)
				if err == nil && ok {
					err = m.repo.HeartbeatTicketRun(leaseCtx, m.Run.ID)
				}
				stop()
				latest := m.s.enabledConfig()
				if err != nil || !ok || latest == nil || latest.Generation != m.cfg.Generation {
					cancel()
					return
				}
			}
		}
	}()
	defer func() { cancel(); <-renewDone }()
	disconnected := !emit("start", m.Run)
	if disconnected {
		cancel()
	}
	events := make(chan CodexTicketAttempt, 8)
	jobs := make(chan struct {
		id    int64
		model string
	})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				m.collectModel(ctx, job.id, job.model, func(e CodexTicketAttempt) { events <- e })
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, id := range m.ids {
			for _, model := range m.cfg.models() {
				jobs <- struct {
					id    int64
					model string
				}{id, model}
			}
		}
	}()
	go func() { wg.Wait(); close(events) }()
	storageFailed := false
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for events != nil {
		select {
		case <-ticker.C:
			if !disconnected && !emit("heartbeat", map[string]string{"run_id": m.Run.ID}) {
				disconnected = true
				cancel()
			}
		case e, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			// 历史存储故障后取消并排空 worker，避免每个未执行账号继续等待数据库超时。
			if storageFailed {
				continue
			}
			writeCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
			err := m.repo.AppendTicketEvent(writeCtx, m.Run.ID, &e)
			stop()
			if err != nil {
				storageFailed = true
				cancel()
			} else {
				if e.Kind == "result" {
					m.Run.Counts[e.Status]++
				}
				if !disconnected && !emit(e.Kind, e) {
					disconnected = true
					cancel()
				}
			}
		}
	}
	m.Run.Status = "completed"
	if ctx.Err() != nil {
		m.Run.Status = "cancelled"
	}
	if storageFailed {
		m.Run.Status = "failed"
	}
	now := time.Now().UTC()
	m.Run.FinishedAt = &now
	finishCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
	err := m.repo.FinishTicketRun(finishCtx, m.Run.ID, m.Run.Status, m.Run.Counts)
	stop()
	if err != nil {
		m.Run.Status = "failed"
	}
	if !disconnected {
		emit("complete", m.Run)
	}
}

func (m *CodexTicketManualSession) collectModel(ctx context.Context, id int64, model string, record func(CodexTicketAttempt)) {
	ctx = context.WithValue(ctx, codexTicketManualContextKey{}, true)
	started := time.Now().UTC()
	latestKey := ""
	result := CodexTicketAttempt{Kind: "result", AccountID: id, Model: model, TargetLength: m.cfg.targetLength(), Status: "skipped", Reason: "ineligible", StartedAt: started}
	defer func() {
		result.FinishedAt = time.Now().UTC()
		result.DurationMS = time.Since(started).Milliseconds()
		m.s.recordLatest(ctx, m.cfg, latestKey, "manual", result)
		record(result)
	}()
	if ctx.Err() != nil {
		result.Status = "cancelled"
		result.Reason = "cancelled"
		return
	}
	a, err := m.s.gateway.accountRepo.GetByID(ctx, id)
	if err != nil || a == nil {
		return
	}
	if codexTicketAccount(a) {
		latestKey = codexTicketKey(m.cfg, a, model, a.GetOpenAIAccessToken())
	}
	result.AccountName = a.Name
	result.Email = firstStringValue(a.Credentials, "email")
	if result.Email == "" {
		result.Email = firstStringValue(a.Extra, "email", "email_address")
	}
	if !codexTicketCollectionAllowed(ctx, a) || !a.IsModelSupported(model) {
		return
	}
	token, _, err := m.s.gateway.GetAccessToken(ctx, a)
	if err != nil || token == "" {
		result.Status = "failed"
		result.Reason = "credential"
		return
	}
	key := codexTicketKey(m.cfg, a, model, token)
	latestKey = key
	previousRaw, err := m.s.cache.Get(ctx, "status:"+key)
	if err != nil {
		result.Status = "failed"
		result.Reason = "storage"
		return
	}
	var previous codexTicketObservation
	_ = json.Unmarshal([]byte(previousRaw), &previous)
	if previous.Diagnostic != nil && previous.Diagnostic.RetryNotBefore != nil && time.Now().Before(*previous.Diagnostic.RetryNotBefore) {
		result.Reason = "backoff"
		result.Diagnostic = safeCodexTicketDiagnostic(previous.Diagnostic)
		return
	}
	// 手动显式重采允许绕过有效票/普通采集周期，但绝不绕过上游 Retry-After。
	previousID := ""
	if previous.Diagnostic != nil {
		previousID = previous.Diagnostic.ProxyID
	}
	failed := previous.State == "failed" || previous.State == "missing"
	for attempt := 1; attempt <= m.cfg.attempts(); attempt++ {
		latest := m.s.enabledConfig()
		if ctx.Err() != nil || latest == nil || latest.Generation != m.cfg.Generation {
			result.Status = "cancelled"
			result.Reason = "cancelled"
			return
		}
		proxy, ok := selectCodexTicketProxy(m.cfg, previousID, failed)
		if !ok {
			result.Status = "failed"
			result.Reason = "proxy_config"
			return
		}
		var event CodexTicketAttempt
		ready, retry := m.s.probeAttempt(ctx, m.cfg, a, model, token, key, proxy, attempt, func(e CodexTicketAttempt) { event = e })
		event.Kind = "attempt"
		event.AccountID = id
		event.AccountName = result.AccountName
		event.Email = result.Email
		event.Model = model
		event.TargetLength = m.cfg.targetLength()
		if event.Diagnostic != nil {
			event.Diagnostic.ProxyName = proxy.Name
		}
		m.s.collectReferenceIP(ctx, proxy, &event)
		record(event)
		result.Status = event.Status
		result.Reason = event.Reason
		result.Attempt = attempt
		result.Diagnostic = event.Diagnostic
		result.ExpiresAt = event.ExpiresAt
		result.ReferenceIP = event.ReferenceIP
		result.IPStatus = event.IPStatus
		result.IPCheckedAt = event.IPCheckedAt
		result.IPSource = event.IPSource
		result.IPHTTPStatus = event.IPHTTPStatus
		if ready || !retry || attempt == m.cfg.attempts() {
			return
		}
		previousID, failed = proxy.ID, true
		timer := time.NewTimer(m.cfg.retryInterval())
		select {
		case <-ctx.Done():
			timer.Stop()
			result.Status = "cancelled"
			result.Reason = "cancelled"
			return
		case <-timer.C:
		}
	}
}
