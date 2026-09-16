package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 复现 HTTP/SSE 客户端在部分输出后取消的场景；上游终态仍须排水用于结算。
func TestForwardOpenAIWSV2_ClientCancellationDrainsWithoutSyntheticFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writer := &cancelOnFirstWriteResponseWriter{cancel: cancel}
	c, _ := gin.CreateTestContext(writer)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil).WithContext(ctx)
	c.Request.Header.Set("User-Agent", "unit-test-agent/1.0")

	captureConn := &openAIWSCancelSafeConn{openAIWSCaptureConn: &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.created","response":{"id":"resp_cancel_1","model":"gpt-5.5"}}`),
			[]byte(`{"type":"response.output_text.delta","delta":"partial"}`),
			[]byte(`{"type":"response.completed","response":{"id":"resp_cancel_1","model":"gpt-5.5","usage":{"input_tokens":3,"output_tokens":5}}}`),
		},
		readDelays: []time.Duration{0, 0, 50 * time.Millisecond},
	}}

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
	cfg.Gateway.OpenAIWS.IngressModeDefault = OpenAIWSIngressModeCtxPool
	cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 1
	cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
	cfg.Gateway.OpenAIWS.MaxIdlePerAccount = 1
	cfg.Gateway.OpenAIWS.QueueLimitPerConn = 8
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 5
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(&openAIWSClientConnCancelDialer{conn: captureConn})
	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     &httpUpstreamRecorder{},
		cache:            &stubGatewayCache{},
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
		openaiWSPool:     pool,
	}
	account := &Account{
		ID:          9101,
		Name:        "openai-ws-client-cancel",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra: map[string]any{
			"openai_apikey_responses_websockets_v2_mode": OpenAIWSIngressModeCtxPool,
		},
	}

	result, err := svc.Forward(ctx, c, account, []byte(`{"model":"gpt-5.5","stream":true,"input":[{"type":"input_text","text":"hello"}]}`))

	require.NoError(t, err, "client cancellation must not surface as an upstream failure")
	require.NotNil(t, result)
	require.True(t, result.ClientDisconnect)
	require.Equal(t, "resp_cancel_1", result.RequestID)
	require.Equal(t, 3, result.Usage.InputTokens)
	require.Equal(t, 5, result.Usage.OutputTokens)
	require.NotContains(t, writer.body.String(), "response.failed")
}

// 取消发生在阻塞读取期间时，租约必须仍可用于读取后续终态，而不是先被淘汰。
func TestForwardOpenAIWSV2_CancelDuringReadDrainsTerminal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil).WithContext(ctx)
	conn := &cancelDuringReadConn{openAIWSCancelSafeConn: &openAIWSCancelSafeConn{openAIWSCaptureConn: &openAIWSCaptureConn{events: [][]byte{
		[]byte(`{"type":"response.created","response":{"id":"cancel-during-read"}}`),
		[]byte(`{"type":"response.completed","response":{"id":"cancel-during-read","usage":{"input_tokens":7,"output_tokens":9}}}`),
	}}}, cancel: cancel}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
	cfg.Gateway.OpenAIWS.IngressModeDefault = OpenAIWSIngressModeCtxPool
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 2
	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(&openAIWSClientConnCancelDialer{conn: conn})
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: &httpUpstreamRecorder{}, cache: &stubGatewayCache{}, openaiWSResolver: NewOpenAIWSProtocolResolver(cfg), toolCorrector: NewCodexToolCorrector(), openaiWSPool: pool}
	account := &Account{ID: 9102, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "test"}, Extra: map[string]any{"openai_apikey_responses_websockets_v2_mode": OpenAIWSIngressModeCtxPool}}
	result, err := svc.Forward(ctx, c, account, []byte(`{"model":"gpt-5.5","stream":true,"input":"hello"}`))
	require.NoError(t, err)
	require.True(t, result.ClientDisconnect)
	require.Equal(t, 7, result.Usage.InputTokens)
	require.Equal(t, 9, result.Usage.OutputTokens)
	require.NotContains(t, recorder.Body.String(), "response.failed")
}

type cancelDuringReadConn struct {
	*openAIWSCancelSafeConn
	cancel context.CancelFunc
	reads  int
}

func (c *cancelDuringReadConn) ReadMessage(ctx context.Context) ([]byte, error) {
	c.reads++
	if c.reads == 2 {
		c.cancel()
		return nil, context.Canceled
	}
	return c.openAIWSCancelSafeConn.ReadMessage(ctx)
}

type cancelOnFirstWriteResponseWriter struct {
	mu     sync.Mutex
	header http.Header
	body   bytes.Buffer
	cancel context.CancelFunc
	writes int
	status int
}

func (w *cancelOnFirstWriteResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *cancelOnFirstWriteResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *cancelOnFirstWriteResponseWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.writes == 0 && w.cancel != nil {
		w.cancel()
	}
	w.writes++
	return w.body.Write(p)
}

func (w *cancelOnFirstWriteResponseWriter) Flush() {}

type openAIWSClientConnCancelDialer struct {
	conn openAIWSClientConn
}

func (d *openAIWSClientConnCancelDialer) Dial(
	ctx context.Context,
	wsURL string,
	headers http.Header,
	proxyURL string,
	_ *tlsfingerprint.Profile,
) (openAIWSClientConn, int, http.Header, error) {
	return d.conn, 0, nil, nil
}

// 读上下文取消时保留延迟事件，使脱离取消信号的排水读可以继续消费终态。
type openAIWSCancelSafeConn struct {
	*openAIWSCaptureConn
}

func (c *openAIWSCancelSafeConn) ReadMessage(ctx context.Context) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, errOpenAIWSConnClosed
	}
	if len(c.events) == 0 {
		c.mu.Unlock()
		return nil, io.EOF
	}
	delay := time.Duration(0)
	if len(c.readDelays) > 0 {
		delay = c.readDelays[0]
	}
	event := c.events[0]
	c.mu.Unlock()
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, errOpenAIWSConnClosed
	}
	if len(c.events) == 0 {
		return nil, io.EOF
	}
	c.events = c.events[1:]
	if len(c.readDelays) > 0 {
		c.readDelays = c.readDelays[1:]
	}
	return event, nil
}
