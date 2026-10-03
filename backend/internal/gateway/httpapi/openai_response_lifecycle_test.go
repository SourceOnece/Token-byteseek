package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	wire "github.com/TokenFlux/TokenRouter/internal/protocol/openai"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// TestResponsesLifecycleCompatibility 覆盖实测裸终态、旧 done、重复用量及真正断流的客户端表现。
func TestResponsesLifecycleCompatibility(t *testing.T) {
	const delta = "event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"Hello!\"}\n\n"
	const completed = `{"object":"response","id":"resp_compat","model":"upstream-model","status":"completed","output":[{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Hello!","annotations":[]}]}],"usage":{"input_tokens":7,"output_tokens":9},"vendor":{"opaque":"keep"}}`
	frame := func(eventType, payload string) string {
		if eventType == "" {
			return "data: " + payload + "\n\n"
		}
		return "event: " + eventType + "\ndata: " + payload + "\n\n"
	}
	tests := []struct {
		name, body string
		types      []string
		wantError  bool
		output     int
	}{
		{
			name: "bare lifecycle followed by duplicate done",
			body: frame("response.created", `{"object":"response","id":"resp_compat","status":"in_progress","output":[]}`) +
				frame("response.in_progress", `{"object":"response","id":"resp_compat","status":"in_progress","output":[]}`) + delta +
				frame("response.completed", strings.Replace(completed, `"output_tokens":9`, `"output_tokens":3`, 1)) +
				frame("response.done", `{"type":"response.done","response":`+completed+`}`),
			types:  []string{"response.created", "response.in_progress", "response.output_text.delta", "response.completed"},
			output: 9,
		},
		{
			name:  "done only",
			body:  frame("response.done", `{"type":"response.done","response":`+completed+`}`),
			types: []string{"response.completed"}, output: 9,
		},
		{
			name:  "data only done",
			body:  delta + frame("", `{"type":"response.done","response":`+completed+`}`),
			types: []string{"response.output_text.delta", "response.completed"}, output: 9,
		},
		{
			name:  "standard completed",
			body:  delta + frame("response.completed", `{"type":"response.completed","sequence_number":10,"response":`+completed+`}`),
			types: []string{"response.output_text.delta", "response.completed"}, output: 9,
		},
		{
			name:  "failed done after output",
			body:  delta + frame("response.done", `{"type":"response.done","response":{"id":"resp_compat","status":"failed","error":{"code":"server_error","message":"upstream failed"},"usage":{"input_tokens":7,"output_tokens":9}}}`),
			types: []string{"response.output_text.delta", "response.failed"}, wantError: true, output: 9,
		},
		{
			name:  "incomplete done",
			body:  delta + frame("response.done", `{"type":"response.done","response":{"id":"resp_compat","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[],"usage":{"input_tokens":7,"output_tokens":9}}}`),
			types: []string{"response.output_text.delta", "response.incomplete"}, output: 9,
		},
		{
			name:  "error without status",
			body:  delta + frame("response.done", `{"type":"response.done","response":{"id":"resp_compat","error":{"code":"server_error","message":"upstream failed"},"usage":{"input_tokens":7,"output_tokens":9}}}`),
			types: []string{"response.output_text.delta", "response.failed"}, wantError: true, output: 9,
		},
		{
			name:  "incomplete without status",
			body:  delta + frame("response.done", `{"type":"response.done","response":{"id":"resp_compat","incomplete_details":{"reason":"max_output_tokens"},"output":[],"usage":{"input_tokens":7,"output_tokens":9}}}`),
			types: []string{"response.output_text.delta", "response.incomplete"}, output: 9,
		},
		{
			name: "truncated stream stays failed",
			body: delta, types: []string{"response.output_text.delta"}, wantError: true,
		},
	}
	for _, mode := range []string{"standard", "async", "passthrough"} {
		for _, tt := range tests {
			t.Run(mode+"/"+tt.name, func(t *testing.T) {
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
				options := &wsFixtureOptions{Output: OpenAIResponseOptions{MaxLineSize: openAIResponseDefaultMaxLineSize}}
				if mode == "async" {
					options.Output.StreamDataIntervalTimeout = 30
				}
				svc := newWSFixture(wsFixtureInputs{options: options, corrector: openai.NewCodexToolCorrector()})
				target := &gatewayprovider.ExecutionProvider{Record: provider.Record{LoadLocation: time.LoadLocation, ID: 1, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey}}
				resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(tt.body))}
				var result *openai.StreamingResult
				var err error
				if mode == "passthrough" {
					result, err = openai.ReadPassthroughStreaming(c.Request.Context(), resp, upstream.NewOutputContext(ResponseSink{Writer: c.Writer}), svc.Output.PassthroughOptions(c.Request.Context(), c, target), time.Now(), "client-model", "upstream-model")
				} else {
					result, err = svc.Output.ReadStreamObservation(c.Request.Context(), resp, c, target, time.Now(), "client-model", "upstream-model", "")
				}
				if tt.wantError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				require.NotNil(t, result)
				require.Equal(t, tt.output, result.Usage.OutputTokens)
				assertOpenAISSEFrames(t, recorder.Body.String(), tt.types)
				// 用客户端实际收到的 JSON type 判定完成，不借助服务端宽松的 event 行回退。
				wire.ForEachOpenAISSEFrame(recorder.Body.String(), func(_ string, data []byte) {
					if gjson.GetBytes(data, "type").String() == "response.completed" {
						require.Equal(t, "completed", gjson.GetBytes(data, "response.status").String())
						require.Equal(t, "client-model", gjson.GetBytes(data, "response.model").String())
						require.Equal(t, "keep", gjson.GetBytes(data, "response.vendor.opaque").String())
						require.True(t, gjson.GetBytes(data, "sequence_number").Exists())
					}
				})
			})
		}
	}
}

// TestResponsesLifecycleErrorThenFailedDone 验证前置 error 不会抑制规范化后的失败终态。
func TestResponsesLifecycleErrorThenFailedDone(t *testing.T) {
	for _, mode := range []string{"standard", "async", "passthrough"} {
		for _, header := range []string{"", "event: response.done\n"} {
			t.Run(mode+"/"+header, func(t *testing.T) {
				body := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hi\"}\n\n" +
					"event: error\ndata: {\"type\":\"error\",\"error\":{\"code\":\"server_error\",\"message\":\"failed\"}}\n\n" +
					header + "data: {\"type\":\"response.done\",\"response\":{\"id\":\"resp_review\",\"status\":\"failed\",\"error\":{\"code\":\"server_error\",\"message\":\"failed\"},\"usage\":{\"input_tokens\":2,\"output_tokens\":1}}}\n\n"
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
				options := &wsFixtureOptions{Output: OpenAIResponseOptions{MaxLineSize: openAIResponseDefaultMaxLineSize}}
				if mode == "async" {
					options.Output.StreamDataIntervalTimeout = 30
				}
				svc := newWSFixture(wsFixtureInputs{options: options, corrector: openai.NewCodexToolCorrector()})
				target := &gatewayprovider.ExecutionProvider{Record: provider.Record{LoadLocation: time.LoadLocation, ID: 1, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}}
				resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
				var result *openai.StreamingResult
				var err error
				if mode == "passthrough" {
					result, err = openai.ReadPassthroughStreaming(c.Request.Context(), resp, upstream.NewOutputContext(ResponseSink{Writer: c.Writer}), svc.Output.PassthroughOptions(c.Request.Context(), c, target), time.Now(), "model", "model")
				} else {
					result, err = svc.Output.ReadStreamObservation(c.Request.Context(), resp, c, target, time.Now(), "model", "model", "")
				}
				require.ErrorContains(t, err, "failed")
				require.NotNil(t, result)
				require.Equal(t, 1, result.Usage.OutputTokens)
				assertOpenAISSEFrames(t, recorder.Body.String(), []string{"response.output_text.delta", "response.failed"})
			})
		}
	}
}

// 尾部用量稍晚到达也应更新结算，客户端只收到一个成功终态。
func TestResponsesLifecycleDelayedTerminalUsage(t *testing.T) {
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = writer.Close() }()
		_, _ = io.WriteString(writer, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"late\",\"status\":\"completed\",\"output\":[],\"usage\":{\"input_tokens\":7,\"output_tokens\":3}}}\n\n")
		// 补充用量晚于一秒，结算取最新值，客户端收到一次完成事件。
		time.Sleep(1200 * time.Millisecond)
		_, _ = io.WriteString(writer, "data: {\"type\":\"response.done\",\"response\":{\"id\":\"late\",\"status\":\"completed\",\"output\":[],\"usage\":{\"input_tokens\":7,\"output_tokens\":9}}}\n\n")
	}()
	recorder := newOpenAIResponseFlushRecorder()
	result, err := runOpenAIResponseFlushTest(recorder, reader, OpenAIResponseOptions{})
	require.NoError(t, err)
	require.Equal(t, 9, result.Usage.OutputTokens)
	body, _ := recorder.snapshot()
	require.Equal(t, 1, strings.Count(body, `"type":"response.completed"`))
	<-done
}
