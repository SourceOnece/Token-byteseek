package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// 常驻读取的实际队列路径也必须保留待排水帧，而非只在 mock 直读时工作。
func TestOpenAIWSDrainableReaderLoopCancellationPreservesConnection(t *testing.T) {
	conn := newOpenAIWSConn("drain-queue", 1, &openAIWSFakeConn{}, nil, nil, "")
	conn.readerLoopResults = make(chan []byte, 1)
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), openAIWSDrainableReadContextKey{}, true))
	cancel()
	_, err := conn.readMessage(ctx)
	require.ErrorIs(t, err, context.Canceled)
	select {
	case <-conn.closedCh:
		t.Fatal("取消不应关闭排水连接")
	default:
	}
	conn.readerLoopResults <- []byte(`{"type":"response.completed"}`)
	data, err := conn.readMessageWithContextTimeout(context.Background(), time.Second)
	require.NoError(t, err)
	require.Contains(t, string(data), "response.completed")
}

// 排水预算超时仍须回收真实连接，避免取消豁免演变成资源泄漏。
func TestOpenAIWSDrainableReaderLoopDeadlineStillCloses(t *testing.T) {
	conn := newOpenAIWSConn("drain-timeout", 1, &openAIWSFakeConn{}, nil, nil, "")
	conn.readerLoopResults = make(chan []byte, 1)
	ctx := context.WithValue(context.Background(), openAIWSDrainableReadContextKey{}, true)
	_, err := conn.readMessageWithContextTimeout(ctx, time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	select {
	case <-conn.closedCh:
	default:
		t.Fatal("排水超时必须回收连接")
	}
}
