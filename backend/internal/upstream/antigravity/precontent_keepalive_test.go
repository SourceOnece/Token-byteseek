package antigravity

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/protocol/bridge"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/stretchr/testify/require"
)

type preContentSink struct {
	executionSink
	ping chan struct{}
	once sync.Once
}

func (s *preContentSink) Emit(event upstream.OutputEvent) error {
	err := s.executionSink.Emit(event)
	if strings.Contains(string(event.Data), ": ping") {
		s.once.Do(func() { close(s.ping) })
	}
	return err
}

// 心跳之后的空流/读错必须终结当前 SSE，不能切号或伪造成功完成。
func TestPreContentKeepaliveLocksFailoverBoundary(t *testing.T) {
	for _, responseAPI := range []bool{false, true} {
		for _, readFailure := range []bool{false, true} {
			sink := &preContentSink{ping: make(chan struct{})}
			output := upstream.NewOutputContext(sink)
			options := executionResponseOptions()
			service := &ResponseAdapter{Options: options}
			var adapter antigravityCompatStreamAdapter = NewAntigravityChatStreamAdapter(output, "fixture", true, options.ReverseTools)
			if responseAPI {
				adapter = NewAntigravityResponsesStreamAdapter(output, "fixture", bridge.ResponsesClientToolMapping{}, options.ReverseTools)
			}
			reader, writer := io.Pipe()
			done := make(chan error, 1)
			go func() {
				_, err := service.handleAntigravityCompatStreamWithTiming(output, &http.Response{Body: reader}, time.Now(), "fixture", adapter, "test", time.Millisecond, time.Second)
				done <- err
			}()
			select {
			case <-sink.ping:
			case <-time.After(time.Second):
				_ = reader.Close()
				t.Fatal("no pre-content heartbeat")
			}
			if readFailure {
				_ = writer.CloseWithError(io.ErrUnexpectedEOF)
			} else {
				_ = writer.Close()
			}
			select {
			case err := <-done:
				require.Error(t, err)
				require.False(t, options.IsFailover(err))
			case <-time.After(time.Second):
				_ = reader.Close()
				t.Fatal("stream did not finish")
			}
			_ = reader.Close()
			require.Contains(t, sink.body.String(), ": ping")
			require.Contains(t, sink.body.String(), "upstream_error")
			require.NotContains(t, sink.body.String(), "response.completed")
			require.NotContains(t, sink.body.String(), "[DONE]")
		}
	}
}

// 尚未发出任何内容的即时空流仍可交给 TokenFlux 原故障切换链。
func TestPreContentEmptyBeforeHeartbeatCanFailover(t *testing.T) {
	sink := &executionSink{}
	output := upstream.NewOutputContext(sink)
	options := executionResponseOptions()
	service := &ResponseAdapter{Options: options}
	adapter := NewAntigravityChatStreamAdapter(output, "fixture", false, options.ReverseTools)
	_, err := service.handleAntigravityCompatStreamWithTiming(output, &http.Response{Body: io.NopCloser(strings.NewReader(""))}, time.Now(), "fixture", adapter, "test", time.Hour, 2*time.Hour)
	require.Error(t, err)
	require.True(t, options.IsFailover(err))
	require.Empty(t, sink.body.String())
}

// 连续注释/心跳不能无限刷新首字等待，绝对期限到达后返回流式错误。
func TestPreContentKeepaliveHasAbsoluteDeadline(t *testing.T) {
	sink := &executionSink{}
	output := upstream.NewOutputContext(sink)
	options := executionResponseOptions()
	service := &ResponseAdapter{Options: options}
	reader, writer := io.Pipe()
	defer func() { _ = reader.Close() }()
	defer func() { _ = writer.Close() }()
	go func() {
		for {
			if _, err := io.WriteString(writer, ": comment\n\n"); err != nil {
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
	adapter := NewAntigravityChatStreamAdapter(output, "fixture", false, options.ReverseTools)
	_, err := service.handleAntigravityCompatStreamWithTiming(output, &http.Response{Body: reader}, time.Now(), "fixture", adapter, "test", time.Millisecond, 30*time.Millisecond)
	require.Error(t, err)
	require.False(t, options.IsFailover(err))
	require.False(t, errors.Is(err, io.EOF))
	require.Contains(t, sink.body.String(), "stream_timeout")
}

// 正常内容迟到但在期限内，先前心跳不影响真实结果、用量与完成事件。
func TestPreContentKeepaliveThenSuccessfulCompletion(t *testing.T) {
	sink := &preContentSink{ping: make(chan struct{})}
	output := upstream.NewOutputContext(sink)
	options := executionResponseOptions()
	service := &ResponseAdapter{Options: options}
	reader, writer := io.Pipe()
	defer func() { _ = reader.Close() }()
	defer func() { _ = writer.Close() }()
	adapter := NewAntigravityChatStreamAdapter(output, "fixture", true, options.ReverseTools)
	type completed struct {
		result *StreamResult
		err    error
	}
	done := make(chan completed, 1)
	go func() {
		result, err := service.handleAntigravityCompatStreamWithTiming(output, &http.Response{Body: reader}, time.Now(), "fixture", adapter, "test", time.Millisecond, time.Second)
		done <- completed{result, err}
	}()
	select {
	case <-sink.ping:
	case <-time.After(time.Second):
		t.Fatal("no heartbeat")
	}
	_, err := io.WriteString(writer, "data: {\"response\":{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"hello\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":3,\"candidatesTokenCount\":1}}}\n\n")
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	select {
	case value := <-done:
		require.NoError(t, value.err)
		require.NotNil(t, value.result)
		require.Equal(t, 3, value.result.Usage.InputTokens)
		require.Equal(t, 1, value.result.Usage.OutputTokens)
	case <-time.After(time.Second):
		t.Fatal("completion did not arrive")
	}
	require.Contains(t, sink.body.String(), "hello")
	require.Contains(t, sink.body.String(), "[DONE]")
}
