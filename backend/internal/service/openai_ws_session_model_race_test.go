package service

import (
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

// 模拟上行 session.update 与下行模型恢复同时执行，race 下不得出现共享字符串竞争。
func TestOpenAIWSSessionModelConcurrentReadWrite(t *testing.T) {
	meta := newOpenAIWSPassthroughUsageMeta("gpt-5.4", nil)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			meta.updateSessionRequestModel([]byte(`{"type":"session.update","session":{"model":"gpt-5.5"}}`))
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_ = meta.requestModelForFrame(nil)
		}
	}()
	wg.Wait()
	require.Equal(t, "gpt-5.5", meta.requestModelForFrame(nil))
	require.Equal(t, "public-alias", meta.requestModelForFrame([]byte(`{"type":"response.create","model":"public-alias"}`)))
}
