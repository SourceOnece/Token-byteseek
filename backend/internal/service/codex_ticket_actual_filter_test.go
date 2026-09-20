//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestCodexTicketActualFilterSyntax(t *testing.T) {
	for _, value := range []string{"", "on", "length:332", "actual_length:356", "rotate,actual_length:356"} {
		require.True(t, ValidCodexTicketFilter(value), value)
	}
	for _, value := range []string{"actual_length:0", "actual_length:9000", "actual_length:356 OR 1=1", "on,off", "length:332,actual_length:356", "on,actual_length:356,off"} {
		require.False(t, ValidCodexTicketFilter(value), value)
	}
}

func putActualObservation(t *testing.T, s *CodexTicketService, cache *ticketCacheStub, a *Account, model string, length int, latest bool) string {
	t.Helper()
	cfg := ticketConfigForAccount(s.config.Load(), a.ID)
	key := codexTicketKey(cfg, a, model, a.GetOpenAIAccessToken())
	d := &CodexTicketDiagnostic{HTTPStatus: 200, HeaderPresent: true, HeaderLength: length, PrefixValid: true}
	var raw []byte
	if latest {
		raw, _ = json.Marshal(CodexTicketLatest{Source: "manual", State: "missing", CheckedAt: time.Now(), Diagnostic: d})
		key = "latest:" + key
	} else {
		raw, _ = json.Marshal(codexTicketObservation{State: "missing", CheckedAt: time.Now(), Diagnostic: d})
		key = "status:" + key
	}
	cache.values[key] = string(raw)
	return key
}

// 目标仍是292，但左侧356必须匹配；新诊断优先、无头不借旧数据，查询完全只读。
func TestCodexTicketActualLengthMatchesDisplayAndIsolation(t *testing.T) {
	s, cache, _ := newTicketTestService()
	enableTicketTest(t, s)
	a := ticketAccount()
	b := ticketAccount()
	b.ID = 2
	putActualObservation(t, s, cache, &a, "gpt-6-astra", 292, false)
	key := putActualObservation(t, s, cache, &a, "gpt-6-astra", 356, true)
	putActualObservation(t, s, cache, &b, "gpt-5.6-sol", 356, false)
	before, _ := json.Marshal(cache.values)
	ids, err := s.filterActualTicketLength(context.Background(), []Account{a, b}, 356)
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{1, 2}, ids)
	ids, err = s.filterActualTicketLength(context.Background(), []Account{a}, 292)
	require.NoError(t, err)
	require.Empty(t, ids)
	after, _ := json.Marshal(cache.values)
	require.Equal(t, before, after)
	require.Empty(t, cache.claims)
	// 最新记录无诊断时，界面不展示旧状态的292，筛选也不能回退。
	raw, _ := json.Marshal(CodexTicketLatest{Source: "auto", State: "failed", CheckedAt: time.Now()})
	cache.values[key] = string(raw)
	ids, err = s.filterActualTicketLength(context.Background(), []Account{a}, 292)
	require.NoError(t, err)
	require.Empty(t, ids)
	putActualObservation(t, s, cache, &a, "gpt-6-astra", 356, true)
	a.Credentials = map[string]any{"access_token": "other-token"}
	ids, err = s.filterActualTicketLength(context.Background(), []Account{a}, 356)
	require.NoError(t, err)
	require.Empty(t, ids)
	cache.fail = true
	_, err = s.filterActualTicketLength(context.Background(), []Account{b}, 356)
	require.Error(t, err)
}

type actualFilterRepo struct {
	AccountRepository
	accounts        []Account
	reads           int
	requestedFilter string
	matched         []int64
}

func (r *actualFilterRepo) ListAllWithFilters(ctx context.Context, _, _, _, _ string, _ int64, _ string) ([]Account, error) {
	r.reads++
	r.requestedFilter = CodexTicketFilter(ctx)
	return r.accounts, nil
}
func (r *actualFilterRepo) ListWithFilters(ctx context.Context, p pagination.PaginationParams, _, _, _, _ string, _ int64, _ string) ([]Account, *pagination.PaginationResult, error) {
	r.matched = CodexTicketMatchedIDs(ctx)
	set := map[int64]bool{}
	for _, id := range r.matched {
		set[id] = true
	}
	rows := []Account{}
	for _, a := range r.accounts {
		if set[a.ID] {
			rows = append(rows, a)
		}
	}
	total := len(rows)
	start := min(p.Offset(), total)
	end := min(start+p.Limit(), total)
	return rows[start:end], &pagination.PaginationResult{Total: int64(total)}, nil
}

func TestCodexTicketActualFilterBeforePaginationAndRequestSnapshot(t *testing.T) {
	s, cache, _ := newTicketTestService()
	enableTicketTest(t, s)
	repo := &actualFilterRepo{}
	for i := 1; i <= 310; i++ {
		a := ticketAccount()
		a.ID = int64(i)
		repo.accounts = append(repo.accounts, a)
	}
	for _, i := range []int{0, 299, 309} {
		putActualObservation(t, s, cache, &repo.accounts[i], "gpt-6-astra", 356, true)
	}
	gateway := &OpenAIGatewayService{}
	gateway.codexTickets.Store(s)
	admin := &adminServiceImpl{accountRepo: repo, runtimeBlocker: gateway}
	ctx := WithCodexTicketFilter(context.Background(), "on,actual_length:356")
	page, total, err := admin.ListAccounts(ctx, 2, 2, "openai", "oauth", "", "", 0, "", "id", "asc")
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, page, 1)
	require.EqualValues(t, 310, page[0].ID)
	require.Equal(t, "on", repo.requestedFilter)
	// 导出/按筛选批量的同次请求分页复用匹配快照，不重复全池读状态。
	_, _, err = admin.ListAccounts(ctx, 1, 2, "openai", "oauth", "", "", 0, "", "id", "asc")
	require.NoError(t, err)
	require.Equal(t, 1, repo.reads)
	newCtx := WithCodexTicketFilter(context.Background(), "on,actual_length:356")
	_, _, err = admin.ListAccounts(newCtx, 1, 2, "openai", "oauth", "", "", 0, "", "id", "asc")
	require.NoError(t, err)
	require.Equal(t, 2, repo.reads)
	cache.fail = true
	_, _, err = admin.ListAccounts(WithCodexTicketFilter(context.Background(), "actual_length:356"), 1, 2, "openai", "oauth", "", "", 0, "", "id", "asc")
	require.Error(t, err)
}
