package qoder

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/protocol"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/stretchr/testify/require"
)

type modelTestClient struct{}

func (modelTestClient) StreamRequestContext(context.Context, *SessionContext, string, []byte, map[string]string) (*http.Response, error) {
	body, _ := json.Marshal(QoderSSEWrapper{Body: `{"choices":[{"delta":{"content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`})
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("data: " + string(body) + "\n\ndata: [DONE]\n\n"))}, nil
}

type modelTestSink struct{ bytes.Buffer }

func (*modelTestSink) Begin(upstream.OutputHead) error     { return nil }
func (s *modelTestSink) Emit(e upstream.OutputEvent) error { _, err := s.Write(e.Data); return err }

// TestQoderDoesNotReportSynthesizedModel 验证三种客户端协议生成的别名不会冒充上游声明。
func TestQoderDoesNotReportSynthesizedModel(t *testing.T) {
	for _, wire := range []protocol.ProtocolID{protocol.ProtocolOpenAIResponses, protocol.ProtocolOpenAIChatCompletions, protocol.ProtocolAnthropicMessages} {
		for _, stream := range []bool{false, true} {
			t.Run(string(wire)+map[bool]string{false: "/json", true: "/stream"}[stream], func(t *testing.T) {
				executor := NewExecutor(ExecuteOptions{})
				target := &Target{ProviderID: 1, Session: func(context.Context) (*SessionContext, error) { return &SessionContext{}, nil }, Client: func() (StreamClient, error) { return modelTestClient{}, nil }}
				sink := &modelTestSink{}
				result, err := executor.Execute(context.Background(), upstream.AttemptInput{Target: target, Protocol: wire, Body: []byte(`{"model":"auto","input":"hi","messages":[{"role":"user","content":"hi"}],"max_tokens":64}`), ResponseModel: "client-alias", Stream: stream}, sink)
				require.NoError(t, err)
				require.Contains(t, sink.String(), "client-alias")
				require.Empty(t, result.UpstreamResponseModel)
			})
		}
	}
}
