package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// 状态单独存储，不包含票据正文、响应正文、代理或凭据，也不加入账号调度投影。
type codexTicketObservation struct {
	State      string                 `json:"state"`
	Reason     string                 `json:"reason,omitempty"`
	CheckedAt  time.Time              `json:"checked_at"`
	ExpiresAt  *time.Time             `json:"expires_at,omitempty"`
	Diagnostic *CodexTicketDiagnostic `json:"diagnostic,omitempty"`
}

type CodexTicketModelStatus struct {
	Latest               *CodexTicketLatest     `json:"latest,omitempty"`
	Model                string                 `json:"model"`
	TargetLength         int                    `json:"target_length"`
	DegradedSignalLength int                    `json:"degraded_signal_length"`
	Blocked              bool                   `json:"blocked"`
	State                string                 `json:"state"`
	Reason               string                 `json:"reason,omitempty"`
	CheckedAt            *time.Time             `json:"checked_at,omitempty"`
	ExpiresAt            *time.Time             `json:"expires_at,omitempty"`
	Diagnostic           *CodexTicketDiagnostic `json:"diagnostic,omitempty"`
}
type CodexTicketAccountStatus struct {
	AccountID        int64                    `json:"account_id"`
	Eligible         bool                     `json:"eligible"`
	CollectionPaused bool                     `json:"collection_paused"`
	Models           []CodexTicketModelStatus `json:"models"`
}
type CodexTicketStatusResponse struct {
	Models     []string                   `json:"models"`
	Enabled    bool                       `json:"enabled"`
	ServerTime time.Time                  `json:"server_time"`
	Items      []CodexTicketAccountStatus `json:"items"`
}

func (s *CodexTicketService) recordObservation(ctx context.Context, key, state, reason string, expires *time.Time, diagnostic ...*CodexTicketDiagnostic) {
	if s.cache == nil || key == "" {
		return
	}
	var detail *CodexTicketDiagnostic
	if len(diagnostic) > 0 {
		detail = diagnostic[0]
	}
	value, err := json.Marshal(codexTicketObservation{State: state, Reason: reason, CheckedAt: time.Now().UTC(), ExpiresAt: expires, Diagnostic: detail})
	if err != nil {
		return
	}
	// 显示写入只占有界预算，取消探测也能结束“采集中”；不控制业务或采集成功与否。
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 150*time.Millisecond)
	defer cancel()
	_ = s.cache.Set(writeCtx, "status:"+key, string(value), 24*time.Hour)
}

// Status 仅访问数据库与缓存，不刷新令牌、不发起探测、不改调度；上限与 HTTP 入口一致。
// @project-doc docs/interfaces/codex_ticket.md#account_status
func (s *CodexTicketService) Status(ctx context.Context, ids []int64) (*CodexTicketStatusResponse, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return nil, errors.New("每次需要 1–100 个账号")
	}
	unique := make([]int64, 0, len(ids))
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, errors.New("账号 ID 必须为正整数")
		}
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cfg, err := s.readConfig(readCtx)
	if err != nil {
		return nil, err
	}
	if s.gateway == nil || s.gateway.accountRepo == nil {
		return nil, errors.New("账号服务不可用")
	}
	accounts, err := s.gateway.accountRepo.GetByIDs(readCtx, unique)
	if err != nil {
		return nil, errors.New("读取账号失败")
	}
	byID := make(map[int64]*Account, len(accounts))
	for _, a := range accounts {
		if a != nil {
			byID[a.ID] = a
		}
	}
	keys := make([]string, 0, len(accounts)*len(cfg.models())*3)
	if cfg.Enabled {
		for _, a := range accounts {
			if !codexTicketAccount(a) {
				continue
			}
			for _, model := range cfg.models() {
				key := codexTicketKey(cfg, a, model, a.GetOpenAIAccessToken())
				keys = append(keys, key, "status:"+key, "latest:"+key)
			}
		}
	}
	values := map[string]string{}
	cacheOK := s.cache != nil && s.cipher != nil
	if len(keys) > 0 && cacheOK {
		values, err = s.cache.GetMany(readCtx, keys)
		cacheOK = err == nil
	}
	now := time.Now().UTC()
	response := &CodexTicketStatusResponse{Enabled: cfg.Enabled, ServerTime: now, Items: make([]CodexTicketAccountStatus, 0, len(unique))}
	response.Models = append([]string(nil), cfg.models()...)
	for _, id := range unique {
		a := byID[id]
		if a == nil {
			continue
		}
		row := CodexTicketAccountStatus{AccountID: id, Eligible: codexTicketAccount(a), Models: []CodexTicketModelStatus{}}
		if row.Eligible {
			row.CollectionPaused = !a.IsSchedulable()
			for _, model := range cfg.models() {
				status := CodexTicketModelStatus{Model: model, State: "pending"}
				switch {
				case !cfg.Enabled:
					status.State = "disabled"
				case !a.IsModelSupported(model):
					status.State = "unsupported"
				case !cacheOK:
					status.State = "unavailable"
				default:
					key := codexTicketKey(cfg, a, model, a.GetOpenAIAccessToken())
					status = s.ticketModelStatus(model, values[key], values["status:"+key], now, cfg.targetLength())
					var latest CodexTicketLatest
					if json.Unmarshal([]byte(values["latest:"+key]), &latest) == nil {
						status.Latest = safeTicketLatest(latest)
					}
					if status.Latest != nil && status.Latest.Diagnostic != nil {
						for _, p := range cfg.proxies() {
							if p.ID == status.Latest.Diagnostic.ProxyID {
								status.Latest.Diagnostic.ProxyName = p.Name
								break
							}
						}
					}
					if status.Diagnostic != nil {
						status.Diagnostic.ProxyName = ""
						for _, p := range cfg.proxies() {
							if p.ID == status.Diagnostic.ProxyID {
								status.Diagnostic.ProxyName = p.Name
								break
							}
						}
					}
					if row.CollectionPaused && status.State == "pending" {
						status.State = "paused"
					}
				}
				// 模型门控独立于账号总开关和质量标签，状态不可读时也不伪装可用。
				status.Blocked = cfg.Enabled && a.IsModelSupported(model) && status.State != "ready"
				status.TargetLength = cfg.targetLength()
				status.DegradedSignalLength = cfg.DegradedSignalLength
				row.Models = append(row.Models, status)
			}
		}
		response.Items = append(response.Items, row)
	}
	return response, nil
}

func (s *CodexTicketService) ticketModelStatus(model, encrypted, observation string, now time.Time, lengths ...int) CodexTicketModelStatus {
	status := CodexTicketModelStatus{Model: model, State: "pending"}
	var record codexTicketObservation
	if observation != "" && json.Unmarshal([]byte(observation), &record) == nil && !record.CheckedAt.IsZero() {
		status.CheckedAt = &record.CheckedAt
		status.Diagnostic = safeCodexTicketDiagnostic(record.Diagnostic)
		switch record.State {
		case "collecting":
			if now.Sub(record.CheckedAt) < time.Minute {
				status.State = "collecting"
			}
		case "failed", "missing":
			status.State = record.State
		case "ready":
			status.State = "missing"
			if record.ExpiresAt != nil && !now.Before(*record.ExpiresAt) {
				status.State = "expired"
			}
		}
		// 仅回传固定原因码，缓存中的任意文本不可进入管理界面。
		switch record.Reason {
		case "network", "upstream", "invalid_ticket", "credential", "storage", "cancelled", "proxy_config":
			status.Reason = record.Reason
		}
	}
	if encrypted != "" {
		raw, err := s.cipher.Decrypt(encrypted)
		if err != nil {
			status.State = "unavailable"
			return status
		}
		var ticket codexTicketValue
		if json.Unmarshal([]byte(raw), &ticket) != nil {
			status.State = "unavailable"
			return status
		}
		if validCodexTicket(ticket, lengths...) {
			status.State = "ready"
			status.ExpiresAt = &ticket.ExpiresAt
		} else if !now.Before(ticket.ExpiresAt) {
			status.State = "expired"
		}
	}
	return status
}
