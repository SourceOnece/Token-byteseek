package openai

import (
	"errors"
	"fmt"
	"github.com/TokenFlux/TokenRouter/internal/infra/telemetry/logging"
	"github.com/TokenFlux/TokenRouter/internal/pkg/logredact"
	"github.com/tidwall/gjson"
	"time"
)

// 控制帧消费能力由连接实现声明，池据此维持空闲连接的读循环。
type openAIWSReaderLoopCapable interface{ RequiresReaderLoop() bool }
type openAIWSUpstreamPingCounter interface{ UpstreamPingCount() int64 }
type openAIWSForceCloser interface{ CloseNow() error }
type openAIWSDrainableReadContextKey struct{}

var errOpenAIWSPoolChanged = errors.New("websocket pool changed")
var errOpenAIWSPreferredConnUnavailable = ErrOpenAIWSPreferredConnUnavailable

type openAIWSAcquireQueueWait struct {
	queued  bool
	rewoken bool
	total   time.Duration
}

const WSConnHealthCheckTO = openAIWSConnHealthCheckTO
const openAIWSLogValueMaxLen = 256

func logOpenAIWSModeInfo(format string, args ...any)  { logging.S().Infof(format, args...) }
func logOpenAIWSModeWarn(format string, args ...any)  { logging.S().Warnf(format, args...) }
func logOpenAIWSModeDebug(format string, args ...any) { logging.S().Debugf(format, args...) }
func truncateOpenAIWSLogValue(value string, limit int) string {
	// 网络错误可能包含完整代理和查询凭据，先脱敏再截断。
	value = logredact.RedactText(logredact.SanitizeUpstreamQueries(value))
	if limit > 0 && len(value) > limit {
		return value[:limit]
	}
	return value
}
func normalizeOpenAIWSLogValue(value string) string {
	return fmt.Sprintf("%q", truncateOpenAIWSLogValue(value, openAIWSLogValueMaxLen))
}
func effectiveOpenAISSEEventType(raw []byte, fallback string) string {
	if kind := gjson.GetBytes(raw, "type").String(); kind != "" {
		return kind
	}
	return fallback
}

// 从池索引删除不可用连接，实际关闭留给锁外执行，避免阻塞其它账号请求。
func (p *WSConnPool) dropDeadConnLocked(ap *openAIWSProviderPool, conn *WSConn, evicted *[]*WSConn) bool {
	if ap == nil || conn == nil || (!conn.isClosed() && !conn.isUnusable()) {
		return false
	}
	if _, exists := ap.conns[conn.id]; !exists {
		return false
	}
	delete(ap.conns, conn.id)
	delete(ap.pinnedConns, conn.id)
	ap.signalChangedLocked()
	*evicted = append(*evicted, conn)
	return true
}
