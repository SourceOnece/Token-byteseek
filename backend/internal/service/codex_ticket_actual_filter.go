package service

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	infraerrors "github.com/TokenFlux/TokenRouter/internal/pkg/errors"
)

// 管理端读取完整候选池的最近可见长度，再交数据库做Count/排序/分页；不触发采集或调度。
func (s *adminServiceImpl) resolveActualTicketFilter(ctx context.Context, platform, accountType, status, search string, groupID int64, privacyMode string) (context.Context, error) {
	static, length, valid := ParseCodexTicketFilter(CodexTicketFilter(ctx))
	if !valid {
		return ctx, infraerrors.BadRequest("INVALID_TICKET_FILTER", "无效的票据筛选")
	}
	if length == 0 {
		return ctx, nil
	}
	// 票据只属于独立OpenAI OAuth；先排除其它平台，避免无谓读取其凭据和缓存。
	if (platform != "" && platform != PlatformOpenAI) || (accountType != "" && accountType != AccountTypeOAuth) {
		return context.WithValue(ctx, codexTicketMatchedIDsKey{}, []int64{}), nil
	}
	fail := func() (context.Context, error) {
		return ctx, infraerrors.New(http.StatusServiceUnavailable, "TICKET_FILTER_UNAVAILABLE", "实际票据状态暂不可用，请稍后刷新")
	}
	provider, ok := s.runtimeBlocker.(interface{ CodexTicketConfiguration() *CodexTicketService })
	if !ok || provider.CodexTicketConfiguration() == nil {
		return fail()
	}
	state, _ := ctx.Value(codexTicketFilterKey{}).(*codexTicketFilterRequest)
	if state == nil {
		return fail()
	}
	keyBytes, _ := json.Marshal([]any{platform, accountType, status, search, groupID, privacyMode, CodexQualityFilter(ctx)})
	key := string(keyBytes)
	state.mu.Lock()
	defer state.mu.Unlock()
	if ids, exists := state.matches[key]; exists {
		return context.WithValue(ctx, codexTicketMatchedIDsKey{}, ids), nil
	}
	// 有界查询失败时返回错误，不返回截断的“全池”结果；同次导出/按筛选批量复用快照。
	read, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	accounts, err := s.accountRepo.ListAllWithFilters(WithCodexTicketFilter(read, static), PlatformOpenAI, AccountTypeOAuth, status, search, groupID, privacyMode)
	if err != nil {
		return fail()
	}
	ids, err := provider.CodexTicketConfiguration().filterActualTicketLength(read, accounts, length)
	if err != nil || read.Err() != nil {
		return fail()
	}
	state.matches[key] = ids
	return context.WithValue(ctx, codexTicketMatchedIDsKey{}, ids), nil
}

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

func (s *CodexTicketService) filterActualTicketLength(ctx context.Context, accounts []Account, length int) ([]int64, error) {
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
			if !a.IsModelSupported(model) {
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
