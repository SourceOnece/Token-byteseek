package service

import (
	"context"
	"encoding/json"
	"time"
)

// 冷却只约束当前凭据和模型的采集/票据门控，不写账号总调度开关。
type CodexTicketCooldownCache interface {
	RecordTicketCollection(context.Context, string, bool, int, int, int, time.Time) error
}
type CodexTicketCollectionState struct {
	ConsecutiveFailures int        `json:"consecutive_failures"`
	CooldownUntil       *time.Time `json:"cooldown_until,omitempty"`
}

func ticketCollectionState(raw string) CodexTicketCollectionState {
	var data struct {
		Failures int   `json:"failures"`
		Until    int64 `json:"until"`
	}
	_ = json.Unmarshal([]byte(raw), &data)
	state := CodexTicketCollectionState{ConsecutiveFailures: data.Failures}
	if data.Until > time.Now().UnixMilli() {
		at := time.UnixMilli(data.Until).UTC()
		state.CooldownUntil = &at
	}
	return state
}
func (s *CodexTicketService) recordTicketCollection(ctx context.Context, cfg *codexTicketConfig, key string, success bool, reason string) {
	if !success && (reason == "cancelled" || reason == "account_changed" || reason == "concurrency_busy") {
		return
	}
	if !s.ticketConfigCurrent(cfg) {
		return
	}
	if cache, ok := s.cache.(CodexTicketCooldownCache); ok {
		write, stop := context.WithTimeout(context.WithoutCancel(ctx), 150*time.Millisecond)
		defer stop()
		_ = cache.RecordTicketCollection(write, key, success, cfg.FailureThreshold, cfg.cooldownSeconds(), cfg.attempts(), time.Now())
	}
}
