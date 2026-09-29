package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"

	"github.com/TokenFlux/TokenRouter/internal/infra/telemetry/logging"

	forwardcore "github.com/TokenFlux/TokenRouter/internal/gateway/forward"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"

	"github.com/TokenFlux/TokenRouter/internal/scheduler"

	"github.com/TokenFlux/TokenRouter/internal/upstream/qoder"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQoderGatewaySessionHashUsesPreviousResponseID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	h := &QoderCompatibleRuntime{}
	body := []byte(`{"model":"deepseek-v4-pro","previous_response_id":"resp_qoder_123","input":"next"}`)

	require.Equal(t, qoderStickySessionHashFromSeed("resp_qoder_123"), h.qoderSessionHash(c, QoderResponses, body, 42))
}

func TestQoderGatewaySessionHashHeaderPrecedence(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Request.Header.Set("session_id", "header-session")
	body := []byte(`{"model":"deepseek-v4-pro","prompt_cache_key":"body-session","messages":[{"role":"user","content":"hi"}]}`)

	h := &QoderCompatibleRuntime{}
	require.Equal(t, qoderStickySessionHashFromSeed("header-session"), h.qoderSessionHash(c, QoderMessages, body, 42))
}

func TestQoderBindStickySessionsUsesDetachedContextAfterCancel(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()

	cache := &qoderStickyBindCacheStub{}
	h := &QoderCompatibleRuntime{options: QoderCompatibleOptions{Execution: &qoderStickyExecutionStub{cache: cache}}}
	groupID := int64(12)

	h.bindQoderStickySessions(parent, &groupID, "request-session-hash", 99, QoderResponses, &forwardcore.MessagesResult{
		RequestID: "resp_qoder_bind",
	}, nil)

	require.False(t, cache.sawCanceledContext)
	require.True(t, cache.sawDeadline)
	require.Len(t, cache.calls, 2)
	require.Equal(t, qoderStickyBindCall{groupID: 12, sessionHash: "request-session-hash", providerID: 99}, cache.calls[0])
	require.Equal(t, qoderStickyBindCall{groupID: 12, sessionHash: qoderStickySessionHashFromSeed("resp_qoder_bind"), providerID: 99}, cache.calls[1])
}

func TestQoderGatewayShouldRefreshProviderOnlyForUnwrittenAuthErrors(t *testing.T) {
	handler := &QoderCompatibleRuntime{
		options: QoderCompatibleOptions{PlatformAvailable: true, MayRefresh: qoder.MayRefreshAttempt},
	}

	require.True(t, handler.shouldRefreshQoderProvider(&qoder.APIError{StatusCode: http.StatusUnauthorized}, false))
	require.True(t, handler.shouldRefreshQoderProvider(&qoder.APIError{StatusCode: http.StatusForbidden}, false))
	require.False(t, handler.shouldRefreshQoderProvider(&qoder.APIError{StatusCode: http.StatusForbidden, Code: "115"}, false))
	require.False(t, handler.shouldRefreshQoderProvider(&qoder.APIError{StatusCode: http.StatusForbidden, Code: "112"}, false))
	require.False(t, handler.shouldRefreshQoderProvider(&qoder.APIError{StatusCode: http.StatusTooManyRequests}, false))
	require.False(t, handler.shouldRefreshQoderProvider(&qoder.APIError{StatusCode: http.StatusUnauthorized}, true))
	require.False(t, (&QoderCompatibleRuntime{}).shouldRefreshQoderProvider(&qoder.APIError{StatusCode: http.StatusUnauthorized}, false))
}

func TestQoderGatewayFailoverExhaustedUsesLastQoderError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/qoder/v1/chat/completions", nil)

	h := &QoderCompatibleRuntime{options: QoderCompatibleOptions{Errors: QoderErrorPresenter{Describe: gatewayprovider.DescribeQoderError}}}
	wrote := h.writeQoderFailoverExhaustedError(c, QoderChat, false, &qoder.APIError{
		StatusCode: http.StatusTooManyRequests,
		Message:    "agent busy",
	})

	require.True(t, wrote)
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	require.Contains(t, w.Body.String(), `"type":"rate_limit_error"`)
	require.Contains(t, w.Body.String(), `"message":"agent busy"`)
}

func TestQoderGatewayStreamingAwareError_MessagesKeepsGenericSSEError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	SetOpsRequestContext(c, "claude-opus-4-6", true)

	h := &QoderCompatibleRuntime{}
	h.streamingAwareError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", true, QoderMessages)

	body := w.Body.String()
	assert.NotContains(t, body, "event: response.failed")
	assert.Contains(t, body, `"type":"error"`)
	assert.Contains(t, body, `"message":"Upstream request failed"`)
}

func TestQoderGatewayStreamingAwareError_ChatCompletionsStreamingEmitsOpenAIErrorAndDone(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	SetOpsRequestContext(c, "qwen3.7-plus", true)

	h := &QoderCompatibleRuntime{}
	h.streamingAwareError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", true, QoderChat)

	body := w.Body.String()
	assert.NotContains(t, body, "event: response.failed")
	assert.NotContains(t, body, `"type":"error"`)
	assert.Contains(t, body, `data: {"error":{"type":"upstream_error","message":"Upstream request failed"}}`)
	assert.Contains(t, body, "data: [DONE]\n\n")
}

func TestQoderGatewayStreamingAwareError_NonStreamingAfterKeepaliveKeepsJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	SetOpsRequestContext(c, "qwen3.7-plus", false)
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err := c.Writer.WriteString("\n")
	require.NoError(t, err)

	h := &QoderCompatibleRuntime{}
	h.streamingAwareError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", true, QoderChat)

	body := w.Body.Bytes()
	require.True(t, json.Valid(body), "body should remain parseable JSON after keepalive whitespace: %q", string(body))
	assert.NotContains(t, string(body), "data:")
	assert.Contains(t, string(body), `"type":"upstream_error"`)
	assert.Contains(t, string(body), `"message":"Upstream request failed"`)
}

func TestQoderGatewaySubmitUsageRecordIgnoresRequestCancellation(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	reqCtx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil).WithContext(reqCtx)

	done := make(chan error, 1)
	h := &QoderCompatibleRuntime{}
	h.submitUsageRecordTask(c, func(ctx context.Context) {
		select {
		case <-ctx.Done():
			done <- ctx.Err()
		default:
			done <- nil
		}
	})

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("usage task did not run")
	}
}

func TestQoderGatewayRequestCanceledDetectsErrorOrContext(t *testing.T) {
	require.True(t, qoderRequestCanceled(context.Background(), context.Canceled))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.True(t, qoderRequestCanceled(ctx, fmt.Errorf("wrapped upstream error")))

	require.False(t, qoderRequestCanceled(context.Background(), fmt.Errorf("upstream error")))
}

func TestQoderStreamReleaseDoesNotFireOnClientCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	released := make(chan struct{}, 1)

	release := wrapQoderReleaseOnDone(ctx, func() {
		released <- struct{}{}
	}, true)

	cancel()
	select {
	case <-released:
		t.Fatal("stream release should wait for explicit completion")
	case <-time.After(20 * time.Millisecond):
	}

	release()
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("explicit release did not run")
	}
	release()
	select {
	case <-released:
		t.Fatal("release should run at most once")
	case <-time.After(20 * time.Millisecond):
	}
}

func TestQoderNonStreamReleaseStillFiresOnClientCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	released := make(chan struct{}, 1)

	release := wrapQoderReleaseOnDone(ctx, func() {
		released <- struct{}{}
	}, false)
	defer release()

	cancel()
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("non-stream release should run on client cancel")
	}
}

func TestQoderGatewayProviderSlotWaitQueueFullReturnsRateLimitBeforePollingSlot(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/qoder/v1/chat/completions", nil)

	cache := &qoderProviderWaitCacheStub{
		helperConcurrencyCacheStub: &helperConcurrencyCacheStub{},
		providerWaitAllowed:        false,
	}
	h := &QoderCompatibleRuntime{
		concurrencyHelper: NewConcurrencyHelper(scheduler.NewConcurrencyService(cache, scheduler.Diagnostics{
			Logf:  logging.LegacyPrintf,
			Event: logging.Event,
		},
		), SSEPingFormatNone, 0),
	}
	streamStarted := false

	release, err := h.acquireQoderProviderSlotWithWait(c, &providercore.ProviderSnapshot{ID: 77, Concurrency: 1}, &scheduler.ProviderWaitPlan{
		ProviderID:     77,
		MaxConcurrency: 1,
		Timeout:        time.Millisecond,
		MaxWaiting:     2,
	}, false, &streamStarted, nil)

	require.Nil(t, release)
	var waitErr *WaitQueueFullError
	require.ErrorAs(t, err, &waitErr)
	require.Equal(t, "provider", waitErr.SlotType)
	require.Equal(t, 1, cache.providerWaitIncrementCalls)
	require.Equal(t, 2, cache.providerWaitMaxWaiting)
	require.Equal(t, 0, cache.providerAcquireCalls, "full wait queue should reject before polling provider slots")
	require.Equal(t, 0, cache.providerWaitDecrementCalls)
}

func TestQoderGatewayProviderSlotWaitCountDecrementsWhenWaitTimesOut(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/qoder/v1/chat/completions", nil)

	cache := &qoderProviderWaitCacheStub{
		helperConcurrencyCacheStub: &helperConcurrencyCacheStub{providerSeq: []bool{false}},
		providerWaitAllowed:        true,
	}
	h := &QoderCompatibleRuntime{
		concurrencyHelper: NewConcurrencyHelper(scheduler.NewConcurrencyService(cache, scheduler.Diagnostics{
			Logf:  logging.LegacyPrintf,
			Event: logging.Event,
		},
		), SSEPingFormatNone, 0),
	}
	streamStarted := false

	release, err := h.acquireQoderProviderSlotWithWait(c, &providercore.ProviderSnapshot{ID: 78, Concurrency: 1}, &scheduler.ProviderWaitPlan{
		ProviderID:     78,
		MaxConcurrency: 1,
		Timeout:        time.Millisecond,
		MaxWaiting:     3,
	}, false, &streamStarted, nil)

	require.Nil(t, release)
	var concurrencyErr *ConcurrencyError
	require.ErrorAs(t, err, &concurrencyErr)
	require.Equal(t, "provider", concurrencyErr.SlotType)
	require.Equal(t, 1, cache.providerWaitIncrementCalls)
	require.Equal(t, 3, cache.providerWaitMaxWaiting)
	require.Equal(t, 1, cache.providerAcquireCalls)
	require.Equal(t, 1, cache.providerWaitDecrementCalls)
}

func TestQoderGatewayProviderSlotWaitCountDecrementsAfterAcquire(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/qoder/v1/chat/completions", nil)

	cache := &qoderProviderWaitCacheStub{
		helperConcurrencyCacheStub: &helperConcurrencyCacheStub{providerSeq: []bool{true}},
		providerWaitAllowed:        true,
	}
	h := &QoderCompatibleRuntime{
		concurrencyHelper: NewConcurrencyHelper(scheduler.NewConcurrencyService(cache, scheduler.Diagnostics{
			Logf:  logging.LegacyPrintf,
			Event: logging.Event,
		},
		), SSEPingFormatNone, 0),
	}
	streamStarted := false

	release, err := h.acquireQoderProviderSlotWithWait(c, &providercore.ProviderSnapshot{ID: 79, Concurrency: 1}, &scheduler.ProviderWaitPlan{
		ProviderID:     79,
		MaxConcurrency: 1,
		Timeout:        time.Second,
		MaxWaiting:     4,
	}, false, &streamStarted, nil)

	require.NoError(t, err)
	require.NotNil(t, release)
	require.Equal(t, 1, cache.providerWaitIncrementCalls)
	require.Equal(t, 4, cache.providerWaitMaxWaiting)
	require.Equal(t, 1, cache.providerAcquireCalls)
	require.Equal(t, 1, cache.providerWaitDecrementCalls)
	release()
	require.Equal(t, 1, cache.providerReleaseCalls)
}

type qoderProviderWaitCacheStub struct {
	*helperConcurrencyCacheStub

	providerWaitAllowed        bool
	providerWaitIncrementCalls int
	providerWaitDecrementCalls int
	providerWaitMaxWaiting     int
}

func (s *qoderProviderWaitCacheStub) IncrementProviderWaitCount(ctx context.Context, providerID int64, maxWait int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providerWaitIncrementCalls++
	s.providerWaitMaxWaiting = maxWait
	return s.providerWaitAllowed, nil
}

func (s *qoderProviderWaitCacheStub) DecrementProviderWaitCount(ctx context.Context, providerID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providerWaitDecrementCalls++
	return nil
}

type qoderStickyBindCall struct {
	groupID     int64
	sessionHash string
	providerID  int64
}

type qoderStickyBindCacheStub struct {
	calls              []qoderStickyBindCall
	sawCanceledContext bool
	sawDeadline        bool
}

func (s *qoderStickyBindCacheStub) GetSessionProviderID(context.Context, int64, string) (int64, error) {
	return 0, nil
}

func (s *qoderStickyBindCacheStub) SetSessionProviderID(ctx context.Context, groupID int64, sessionHash string, providerID int64, _ time.Duration) error {
	if ctx.Err() != nil {
		s.sawCanceledContext = true
		return ctx.Err()
	}
	if _, ok := ctx.Deadline(); ok {
		s.sawDeadline = true
	}
	s.calls = append(s.calls, qoderStickyBindCall{groupID: groupID, sessionHash: sessionHash, providerID: providerID})
	return nil
}

func (s *qoderStickyBindCacheStub) RefreshSessionTTL(context.Context, int64, string, time.Duration) error {
	return nil
}

func (s *qoderStickyBindCacheStub) DeleteSessionProviderID(context.Context, int64, string) error {
	return nil
}

func (s *qoderStickyBindCacheStub) SetSessionOwnerGroupID(context.Context, int64, string, string, int64, time.Duration) (bool, error) {
	return true, nil
}

func (s *qoderStickyBindCacheStub) GetSessionOwnerGroupID(context.Context, int64, string, string) (int64, error) {
	return 0, nil
}

func (s *qoderStickyBindCacheStub) RefreshSessionOwnerTTL(context.Context, int64, string, string, time.Duration) error {
	return nil
}

// qoderStickyExecutionStub 只记录原生粘性端口调用，不创建旧网关服务。
type qoderStickyExecutionStub struct {
	QoderCompatibleExecution
	cache session.GatewayCache
}

func (s *qoderStickyExecutionStub) BindStickySession(ctx context.Context, id *int64, hash string, providerID int64) error {
	var groupID int64
	if id != nil {
		groupID = *id
	}
	return s.cache.SetSessionProviderID(ctx, groupID, hash, providerID, time.Hour)
}
