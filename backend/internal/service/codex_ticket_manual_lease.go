package service

import (
	"context"
	"encoding/json"
	"sort"
	"time"
)

// 整批手动任务只保留尚未完成的账号；单条租约避免无限任务拖住全池自动续期。
type codexTicketManualLease struct {
	RunID    string  `json:"run_id"`
	Accounts []int64 `json:"accounts"`
}

func (m *CodexTicketManualSession) leaseValueLocked() string {
	ids := make([]int64, 0, len(m.remaining))
	for id := range m.remaining {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	value, _ := json.Marshal(codexTicketManualLease{RunID: m.Run.ID, Accounts: ids})
	return string(value)
}

func (m *CodexTicketManualSession) renewSelection(ctx context.Context, completedID int64) (bool, error) {
	m.leaseMu.Lock()
	defer m.leaseMu.Unlock()
	if completedID > 0 {
		m.remaining[completedID]--
		if m.remaining[completedID] <= 0 {
			delete(m.remaining, completedID)
		}
	}
	replacement := m.leaseValueLocked()
	ok, err := m.s.cache.ReplaceLease(ctx, "manual:"+m.cfg.Generation, m.leaseOwner, replacement, 45*time.Second)
	if ok && err == nil {
		m.leaseOwner = replacement
	}
	return ok, err
}

func (s *CodexTicketService) manualAccountReserved(ctx context.Context, id int64) bool {
	cfg := s.config.Load()
	if cfg == nil || s.cache == nil {
		return true
	}
	read, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	raw, err := s.cache.LeaseValue(read, "manual:"+cfg.Generation)
	if err != nil {
		return true
	}
	if raw == "" {
		return false
	}
	var selection codexTicketManualLease
	if json.Unmarshal([]byte(raw), &selection) != nil || selection.RunID == "" || len(selection.Accounts) > 500 {
		return true
	}
	for _, accountID := range selection.Accounts {
		if accountID == id {
			return true
		}
	}
	return false
}
