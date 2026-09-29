package codexticket

import (
	"context"
	"encoding/json"
	"net/http"

	infraerrors "github.com/TokenFlux/TokenRouter/internal/pkg/apperror"
)

// 与账号列displayDiagnostic保持一致：有效latest优先，即使其diagnostic为空也不回退旧status。
func visibleTicketDiagnostic(latestRaw, observationRaw string) *CodexTicketDiagnostic {
	var latest CodexTicketLatest
	if json.Unmarshal([]byte(latestRaw), &latest) == nil {
		if safe := safeTicketLatest(latest); safe != nil {
			return safe.Diagnostic
		}
	}
	var observation codexTicketObservation
	if json.Unmarshal([]byte(observationRaw), &observation) == nil && !observation.CheckedAt.IsZero() {
		return safeCodexTicketDiagnostic(observation.Diagnostic)
	}
	return nil
}

func (s *CodexTicketService) FilterActualTicketLength(ctx context.Context, accounts []Account, length int) ([]int64, error) {
	cfg, err := s.readConfig(ctx)
	if err != nil {
		return nil, err
	}
	matched := []int64{}
	if !cfg.Enabled {
		return matched, nil
	}
	if s.cache == nil || s.cipher == nil {
		return nil, infraerrors.New(http.StatusServiceUnavailable, "TICKET_FILTER_UNAVAILABLE", "票据缓存不可用")
	}
	type candidate struct {
		id  int64
		key string
	}
	pending := make([]candidate, 0, 256)
	seen := map[int64]bool{}
	flush := func() error {
		if len(pending) == 0 {
			return ctx.Err()
		}
		keys := make([]string, 0, len(pending)*2)
		for _, item := range pending {
			keys = append(keys, "latest:"+item.key, "status:"+item.key)
		}
		values, err := s.cache.GetMany(ctx, keys)
		if err != nil {
			return err
		}
		for _, item := range pending {
			d := visibleTicketDiagnostic(values["latest:"+item.key], values["status:"+item.key])
			if d != nil && d.HTTPStatus != 0 && d.HeaderPresent && d.HeaderLength == length && !seen[item.id] {
				seen[item.id] = true
				matched = append(matched, item.id)
			}
		}
		pending = pending[:0]
		return ctx.Err()
	}
	for i := range accounts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		a := &accounts[i]
		if !codexTicketAccount(a) {
			continue
		}
		ac := ticketConfigForAccount(cfg, a.ID)
		if !ac.Enabled {
			continue
		}
		for _, model := range ac.models() {
			if seen[a.ID] {
				break
			}
			if !s.runtime.TicketModelSupported(a, model) {
				continue
			}
			pending = append(pending, candidate{id: a.ID, key: codexTicketKey(ac, a, model, a.GetOpenAIAccessToken())})
			if len(pending) == 256 {
				if err := flush(); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return matched, nil
}
