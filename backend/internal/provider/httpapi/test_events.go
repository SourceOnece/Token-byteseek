package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// TestStreamWriter 是真实 HTTP 输出边界，业务用例只看到事件接口。
type TestStreamWriter interface {
	http.ResponseWriter
	http.Flusher
}
type TestEventSink struct{ writer TestStreamWriter }

func NewTestEventSink(writer TestStreamWriter) *TestEventSink { return &TestEventSink{writer: writer} }

func (s *TestEventSink) Begin(_ context.Context, commit bool) error {
	s.writer.Header().Set("Content-Type", "text/event-stream")
	s.writer.Header().Set("Cache-Control", "no-cache")
	if commit {
		s.writer.Header().Set("Connection", "keep-alive")
		s.writer.Header().Set("X-Accel-Buffering", "no")
		s.writer.Flush()
	}
	return nil
}

func (s *TestEventSink) Emit(_ context.Context, event provider.TestEvent) error {
	// 保留旧编码失败时的空 data 行，写出失败则返回给用例取消执行。
	raw, _ := json.Marshal(event)
	if _, err := fmt.Fprintf(s.writer, "data: %s\n\n", raw); err != nil {
		return err
	}
	s.writer.Flush()
	return nil
}
