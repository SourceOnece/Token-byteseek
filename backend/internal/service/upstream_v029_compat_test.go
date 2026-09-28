//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/pkg/apicompat"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 模拟等锁成功后重读过程中任务取消，禁止已取消任务仍向上游交换凭据。
type cancelOnReadRefreshRepo struct {
	refreshAPIAccountRepo
	cancel context.CancelFunc
}

func (r *cancelOnReadRefreshRepo) GetByID(ctx context.Context, id int64) (*Account, error) {
	a, err := r.refreshAPIAccountRepo.GetByID(ctx, id)
	r.cancel()
	return a, err
}
func TestV029RefreshCancellationBeforeExchange(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := &Account{ID: 99, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive}
	repo := &cancelOnReadRefreshRepo{refreshAPIAccountRepo: refreshAPIAccountRepo{account: a}, cancel: cancel}
	executor := &refreshAPIExecutorStub{needsRefresh: true}
	api := NewOAuthRefreshAPI(repo, nil)
	_, err := api.RefreshIfNeeded(ctx, a, executor, time.Minute)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, executor.refreshCalls)
	lock := api.getLocalLock(executor.CacheKey(a))
	probe, cancelProbe := context.WithTimeout(context.Background(), time.Second)
	defer cancelProbe()
	require.NoError(t, lock.Lock(probe))
	lock.Unlock()
}

func TestV029DisabledThinkingSurvivesFinalModelMapping(t *testing.T) {
	req := &apicompat.AnthropicRequest{Model: "gpt-6-astra", Thinking: &apicompat.AnthropicThinking{Type: "disabled"}, OutputConfig: &apicompat.AnthropicOutputConfig{Effort: "max"}, Messages: []apicompat.AnthropicMessage{{Role: "user", Content: json.RawMessage(`"hello"`)}}}
	response, err := apicompat.AnthropicToResponses(req)
	require.NoError(t, err)
	require.Equal(t, "none", response.Reasoning.Effort)
	require.Empty(t, response.Reasoning.Summary)
	chat, err := apicompat.AnthropicToChatCompletionsRequest(req)
	require.NoError(t, err)
	require.Equal(t, "none", chat.ReasoningEffort)
	require.Equal(t, "none", openAICompatAnthropicReasoningEffort(req, "gpt-6-astra", chat.ReasoningEffort))
}

func TestV029OpenCodeDeepSeekPlaceholderScope(t *testing.T) {
	for _, tc := range []struct {
		host, model string
		want        bool
	}{
		{"https://opencode.ai/zen/v1", "deepseek-v4-flash", true},
		{"https://opencode.ai/zen/go/v1", "opencode/deepseek-v4-flash", true},
		{"https://opencode.ai/zen/v1", "gpt-6-astra", false},
		{"https://opencode.ai.example/v1", "deepseek-v4-flash", false},
	} {
		a := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": tc.host}}
		body := []byte(`{"model":"` + tc.model + `","messages":[{"role":"assistant","content":"hello"}]}`)
		out := ensureDeepSeekChatReasoningPlaceholders(a, body)
		require.Equal(t, tc.want, gjson.GetBytes(out, "messages.0.reasoning_content").Exists(), tc.host+tc.model)
		if !tc.want {
			require.Equal(t, string(body), string(out))
		}
	}
}

func TestV029AntigravitySignatureOnlyStreamIsNotSuccess(t *testing.T) {
	for _, responses := range []bool{false, true} {
		svc := newAntigravityCompatService(config.GatewayConfig{MaxLineSize: defaultMaxLineSize}, nil)
		c, w := newAntigravityCompatContext(http.MethodPost, "/", nil)
		resp := &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"thoughtSignature\":\"signature-only\"}]},\"finishReason\":\"MALFORMED_FUNCTION_CALL\"}]}}\n\n"))}
		var err error
		if responses {
			_, err = svc.handleResponsesStreamingFromAntigravity(c, resp, time.Now(), "gemini-3.1-pro-high", apicompat.ResponsesClientToolMapping{})
		} else {
			_, err = svc.handleChatCompletionsStreamingFromAntigravity(c, resp, time.Now(), "gemini-3.1-pro-high", true)
		}
		var retry *UpstreamFailoverError
		require.ErrorAs(t, err, &retry)
		require.True(t, retry.RetryableOnSameAccount)
		require.Empty(t, w.Body.String())
	}
}
