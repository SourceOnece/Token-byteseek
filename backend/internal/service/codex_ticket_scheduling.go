package service

import (
	"context"
	"log/slog"
	"time"
)

// 仅明确长度结论调用；仓储用账号版本条件更新调度及outbox，不修改质量检测结果。
// @project-doc docs/interfaces/codex_ticket.md#length_scheduling
type CodexTicketSchedulingRepository interface {
	ApplyCodexTicketScheduling(context.Context, *Account, bool) (time.Time, error)
}

// 手动和自动共用长度结论；旧配置/取消/并发账号修改不能被迟到结果覆盖。
func (s *CodexTicketService) applyTicketScheduling(ctx context.Context, cfg *codexTicketConfig, account *Account, enabled bool) string {
	state, _ := s.applyTicketSchedulingVersion(ctx, cfg, account, enabled)
	return state
}

// 返回本次SQL实际写入的版本；零值代表未更新，不能通过二次读取猜测自己的写入版本。
func (s *CodexTicketService) applyTicketSchedulingVersion(ctx context.Context, cfg *codexTicketConfig, account *Account, enabled bool) (string, time.Time) {
	if ctx.Err() != nil || !s.ticketConfigCurrent(cfg) || !codexTicketCollectionAllowed(ctx, account) {
		return "stale", time.Time{}
	}
	if account.Schedulable == enabled {
		if enabled {
			return "already_on", time.Time{}
		}
		return "already_off", time.Time{}
	}
	repo, ok := s.gateway.accountRepo.(CodexTicketSchedulingRepository)
	if !ok {
		return "failed", time.Time{}
	}
	write, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	version, err := repo.ApplyCodexTicketScheduling(write, account, enabled)
	if err != nil {
		return "failed", time.Time{}
	}
	if version.IsZero() {
		return "stale", time.Time{}
	}
	if enabled {
		return "enabled", version
	}
	return "disabled", version
}

// 每账号仅保留最新长度信号，最多256个待处理账号；无响应正文或明文凭据。
type ticketSchedulingSignal struct {
	receipt codexTicketReceipt
	enabled bool
	seenAt  time.Time
	// 只允许越过同一worker刚完成的精确版本；人工编辑或其它来源更新仍使旧信号失效。
	previousWrite time.Time
}

func (s *CodexTicketService) queueTicketScheduling(r *codexTicketReceipt, enabled bool) {
	if r == nil || r.cfg.accountID <= 0 || !s.ticketConfigCurrent(r.cfg) {
		return
	}
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.stopped {
		return
	}
	s.schedulingMu.Lock()
	if s.schedulingPending == nil {
		s.schedulingPending = make(map[int64]ticketSchedulingSignal)
	}
	_, exists := s.schedulingPending[r.cfg.accountID]
	if !exists && len(s.schedulingPending) >= 256 {
		s.schedulingMu.Unlock()
		if n := s.schedulingDropped.Add(1); n == 1 || n%100 == 0 {
			slog.Warn("票据调度信号队列已满，保持原调度", "dropped", n)
		}
		return
	}
	s.schedulingPending[r.cfg.accountID] = ticketSchedulingSignal{receipt: *r, enabled: enabled, seenAt: time.Now()}
	s.schedulingMu.Unlock()
	select {
	case s.schedulingWake <- struct{}{}:
	default:
	}
}

func (s *CodexTicketService) runTicketScheduling(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.schedulingWake:
		case <-ticker.C:
		}
		for ctx.Err() == nil {
			s.schedulingMu.Lock()
			var item ticketSchedulingSignal
			for id, signal := range s.schedulingPending {
				item = signal
				delete(s.schedulingPending, id)
				break
			}
			s.schedulingMu.Unlock()
			if item.seenAt.IsZero() {
				break
			}
			s.applyTicketSchedulingSignal(ctx, item)
		}
	}
}

// 业务回答不等待数据库；消费时复核实际注入票据、账号版本与配置，不拿旧响应覆盖新票/人工修改。
func (s *CodexTicketService) applyTicketSchedulingSignal(ctx context.Context, signal ticketSchedulingSignal) {
	r := signal.receipt
	if s.gateway == nil || s.gateway.accountRepo == nil || !s.ticketConfigCurrent(r.cfg) {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	a, err := s.gateway.accountRepo.GetByID(ctx, r.cfg.accountID)
	if err != nil || !codexTicketCollectionAllowed(ctx, a) || (a.UpdatedAt.After(signal.seenAt) && !a.UpdatedAt.Equal(signal.previousWrite)) || !a.IsModelSupported(r.model) || codexTicketKey(r.cfg, a, r.model, a.GetOpenAIAccessToken()) != r.key {
		return
	}
	current, err := s.cache.Get(ctx, r.key)
	// 降智守护可能已原子删除本票，允许缺失；新票已取代旧票时两种方向均忽略旧响应。
	if err != nil || (current != "" && current != r.encoded) || (signal.enabled && current != r.encoded) {
		return
	}
	state, version := s.applyTicketSchedulingVersion(ctx, r.cfg, a, signal.enabled)
	if !version.IsZero() {
		// 较新信号可能在旧信号出队至写库之间到达。仅传递数据库返回的确切版本，
		// 不放宽seenAt，也不容许覆盖其后发生的人工修改或质量检测写入。
		s.schedulingMu.Lock()
		if pending, ok := s.schedulingPending[a.ID]; ok && pending.seenAt.After(signal.seenAt) {
			pending.previousWrite = version
			s.schedulingPending[a.ID] = pending
		}
		s.schedulingMu.Unlock()
	}
	if state == "failed" {
		slog.Warn("票据守护调度更新失败，保持原调度", "account_id", a.ID)
	}
}
