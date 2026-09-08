//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func validQualityRequest() CodexQualityRequest {
	return CodexQualityRequest{AccountIDs: []int64{1}, Model: "gpt-6-astra", Prompt: "test", Keyword: "PASS", ConfirmScheduling: true, ReasoningEffort: "high"}
}

func TestCodexQualityRequestValidation(t *testing.T) {
	for name, change := range map[string]func(*CodexQualityRequest){
		"confirmation":  func(r *CodexQualityRequest) { r.ConfirmScheduling = false },
		"empty keyword": func(r *CodexQualityRequest) { r.Keyword = " " },
		"duplicate":     func(r *CodexQualityRequest) { r.AccountIDs = []int64{1, 1} },
		"concurrency":   func(r *CodexQualityRequest) { r.Concurrency = 6 },
		"effort":        func(r *CodexQualityRequest) { r.ReasoningEffort = "invented" },
		"null":          func(r *CodexQualityRequest) { r.Prompt = "\x00" },
	} {
		t.Run(name, func(t *testing.T) { r := validQualityRequest(); change(&r); require.Error(t, r.Normalize()) })
	}
	r := validQualityRequest()
	require.NoError(t, r.Normalize())
	require.Equal(t, 3, r.Concurrency)
}

func TestCodexQualityCompleteAnswerOnly(t *testing.T) {
	for _, tt := range []struct {
		name, body, want string
		fail             bool
	}{
		{"delta", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"PASS\"}\n\ndata: {\"type\":\"response.completed\"}\n\n", "PASS", false},
		{"truncated", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"PASS\"}\n\n", "", true},
		{"failed", "data: {\"type\":\"response.failed\"}\n\n", "", true},
		{"incomplete", "data: {\"type\":\"response.completed\",\"response\":{\"status\":\"incomplete\"}}\n\n", "", true},
		{"final only", "data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"最终 PASS\"}]}]}}", "最终 PASS", false},
		{"reasoning ignored", "data: {\"type\":\"response.reasoning_summary_text.delta\",\"delta\":\"PASS\"}\n\ndata: {\"type\":\"response.completed\"}\n\n", "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/", nil).WithContext(context.WithValue(context.Background(), codexQualityContextKey, &CodexQualityRequest{}))
			err := (&AccountTestService{}).processOpenAIStream(c, strings.NewReader(tt.body))
			if tt.fail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			text, _ := parseTestSSEOutput(w.Body.String())
			require.Equal(t, tt.want, text)
		})
	}
}

type qualityRepoStub struct {
	AccountRepository
	account   *Account
	saved     *CodexQualityResult
	finishErr error
	onGet     func() *Account
}

func (r *qualityRepoStub) GetByID(context.Context, int64) (*Account, error) {
	if r.onGet != nil {
		return r.onGet(), nil
	}
	return r.account, nil
}
func (r *qualityRepoStub) AcquireCodexQualityTest(context.Context, int64, string) (bool, error) {
	return true, nil
}
func (r *qualityRepoStub) FinishCodexQualityTest(_ context.Context, _ *Account, _ string, result *CodexQualityResult) (bool, error) {
	r.saved = result
	if r.finishErr != nil {
		return false, r.finishErr
	}
	result.SchedulingApplied = result.Status != "cancelled" && result.Status != "stale"
	result.Schedulable = result.Status == "full"
	return result.SchedulingApplied, nil
}
func (r *qualityRepoStub) ListCodexQualityResults(context.Context, []int64, bool) ([]*CodexQualityResult, error) {
	return nil, nil
}

func TestCodexQualityRunClassifiesAndSendsEffort(t *testing.T) {
	for _, tt := range []struct {
		name, answer, status string
		httpStatus           int
	}{
		{"match", "PASS", "full", 200}, {"case mismatch", "pass", "degraded", 200}, {"empty", "", "failed", 200}, {"unauthorized", "PASS secret", "failed", 401},
	} {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{ID: 1, Name: "test", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, UpdatedAt: time.Now(), Credentials: map[string]any{"access_token": "private-token", "email": "test@example.invalid"}}
			repo := &qualityRepoStub{account: account}
			delta, _ := json.Marshal(map[string]any{"type": "response.output_text.delta", "delta": tt.answer})
			body := "data: " + string(delta) + "\n\ndata: {\"type\":\"response.completed\"}\n\n"
			up := &queuedHTTPUpstream{responses: []*http.Response{newJSONResponse(tt.httpStatus, body)}}
			svc := &AccountTestService{accountRepo: repo, httpUpstream: up, openAIGatewayService: &OpenAIGatewayService{}}
			opts := validQualityRequest()
			result := svc.RunCodexQualityTest(context.Background(), 1, &opts)
			require.Equal(t, tt.status, result.Status)
			require.Equal(t, tt.status == "full", result.Schedulable)
			require.True(t, result.SchedulingApplied)
			require.Equal(t, "test@example.invalid", result.Email)
			require.Len(t, up.requests, 1)
			requestBody, err := io.ReadAll(up.requests[0].Body)
			require.NoError(t, err)
			require.Contains(t, string(requestBody), `"effort":"high"`)
			require.Contains(t, string(requestBody), `"text":"test"`)
			require.NotContains(t, result.Error, "private-token")
			require.NotContains(t, result.Error, "PASS secret")
		})
	}
}

func TestCodexQualitySkipsOtherAccounts(t *testing.T) {
	for _, account := range []*Account{{Platform: PlatformAnthropic, Type: AccountTypeOAuth}, {Platform: PlatformOpenAI, Type: AccountTypeAPIKey}} {
		repo := &qualityRepoStub{account: account}
		svc := &AccountTestService{accountRepo: repo}
		opts := validQualityRequest()
		result := svc.RunCodexQualityTest(context.Background(), 1, &opts)
		require.Equal(t, "skipped", result.Status)
		require.Nil(t, repo.saved)
	}
}

func TestCodexQualityConfigurationIdentityIgnoresOnlyObservations(t *testing.T) {
	before := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Credentials: map[string]any{"access_token": "first"}, Extra: map[string]any{"codex_5h_used_percent": 1}}
	after := *before
	after.UpdatedAt = time.Now()
	after.Extra = map[string]any{"codex_5h_used_percent": 2}
	require.True(t, sameQualityAccountConfiguration(before, &after))
	after.Schedulable = false
	require.False(t, sameQualityAccountConfiguration(before, &after))
	after.Schedulable = true
	after.Credentials = map[string]any{"access_token": "second"}
	require.False(t, sameQualityAccountConfiguration(before, &after))
	after.Credentials = before.Credentials
	after.Extra["codex_fingerprint_mode"] = "full"
	require.False(t, sameQualityAccountConfiguration(before, &after))
}

func TestCodexQualityBatchLock(t *testing.T) {
	svc := &AccountTestService{}
	require.True(t, svc.BeginCodexQualityBatch())
	require.False(t, svc.BeginCodexQualityBatch())
	svc.EndCodexQualityBatch()
	require.True(t, svc.BeginCodexQualityBatch())
	svc.EndCodexQualityBatch()
}

func TestCodexQualityConfigurationChangedNeverAppliesOldResult(t *testing.T) {
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, UpdatedAt: time.Now(), Credentials: map[string]any{"access_token": "test-token"}}
	other := *account
	other.Proxy = &Proxy{ID: 9}
	count := 0
	repo := &qualityRepoStub{account: account, onGet: func() *Account {
		count++
		if count == 1 {
			return account
		}
		return &other
	}}
	body := `data: {"type":"response.output_text.delta","delta":"PASS"}` + "\n\n" + `data: {"type":"response.completed"}` + "\n\n"
	up := &queuedHTTPUpstream{responses: []*http.Response{newJSONResponse(200, body)}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: up, openAIGatewayService: &OpenAIGatewayService{}}
	opts := validQualityRequest()
	result := svc.RunCodexQualityTest(context.Background(), 1, &opts)
	require.Equal(t, "stale", result.Status)
	require.False(t, result.SchedulingApplied)
}

func TestCodexQualityStorageFailureNeverClaimsSchedulingApplied(t *testing.T) {
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"access_token": "test-token"}}
	repo := &qualityRepoStub{account: account, finishErr: errors.New("storage failed")}
	body := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"PASS\"}\n\ndata: {\"type\":\"response.completed\"}\n\n"
	up := &queuedHTTPUpstream{responses: []*http.Response{newJSONResponse(200, body)}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: up, openAIGatewayService: &OpenAIGatewayService{}}
	opts := validQualityRequest()
	result := svc.RunCodexQualityTest(context.Background(), 1, &opts)
	require.Equal(t, "failed", result.Status)
	require.False(t, result.SchedulingApplied)
}

func TestCodexQualityCancelledNeverEnablesScheduling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"access_token": "test-token"}}
	repo := &qualityRepoStub{account: account}
	svc := &AccountTestService{accountRepo: repo}
	opts := validQualityRequest()
	result := svc.RunCodexQualityTest(ctx, 1, &opts)
	require.Equal(t, "cancelled", result.Status)
	require.False(t, result.SchedulingApplied)
}
