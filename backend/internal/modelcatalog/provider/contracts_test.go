package provider

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCatalogQueryFreezesOneCandidateFactory 验证一次查询只生成一次完整身份候选，不再尝试日期回退。
func TestCatalogQueryFreezesOneCandidateFactory(t *testing.T) {
	factories, lookups := 0, 0
	options := Options{ModelLookupCandidates: func() func(string) []string {
		factories++
		return func(model string) []string {
			lookups++
			if model == "gpt-9.0" {
				return []string{"priced-model"}
			}
			return []string{model}
		}
	}}
	service := NewServiceFromSnapshot(options, nil, Snapshot{Data: map[string]*CatalogModelPricing{"priced-model": {InputCostPerToken: 0.001}}})
	price := service.GetModelPricing("gpt-9.0-20260101")
	require.Nil(t, price)
	require.Equal(t, 1, factories)
	require.Equal(t, 1, lookups)
}

// 快照不暴露可写缓存别名，并保留 nil 与显式空切片。
func TestCatalogSnapshotIsIndependent(t *testing.T) {
	service := NewServiceFromSnapshot(Options{}, nil, Snapshot{Data: map[string]*CatalogModelPricing{"model": {InputCostPerToken: 1, SupportedModalities: []string{"text"}, SupportedOutputModalities: []string{}}}})
	snapshot := service.Snapshot()
	require.NotNil(t, snapshot.Data["model"].SupportedOutputModalities)
	snapshot.Data["model"].InputCostPerToken = 2
	snapshot.Data["model"].SupportedModalities[0] = "image"
	delete(snapshot.Data, "model")
	stored := service.Snapshot().Data["model"]
	require.Equal(t, 1.0, stored.InputCostPerToken)
	require.Equal(t, []string{"text"}, stored.SupportedModalities)
	empty := NewServiceFromSnapshot(Options{}, nil, Snapshot{})
	require.Nil(t, empty.Snapshot().Data)
}

type lifecyclePricingRemote struct{ calls atomic.Int64 }

func (r *lifecyclePricingRemote) FetchCatalog(context.Context, string, string) ([]byte, string, bool, error) {
	r.calls.Add(1)
	return []byte(hotReloadCatalogJSON), "", false, nil
}

// 构造不加载数据；显式初始化后才启动唯一更新任务，重复停止等待同一任务退出。
func TestPricingConstructionAndLifecycle(t *testing.T) {
	remote := &lifecyclePricingRemote{}
	service := NewService(Options{DataDir: t.TempDir(), RemoteURL: "https://pricing.invalid/catalog"}, remote)
	require.Zero(t, remote.calls.Load())
	require.NoError(t, service.Initialize())
	require.Zero(t, remote.calls.Load())
	service.Start()
	service.Start()
	service.Stop()
	service.Stop()
	require.Equal(t, int64(1), remote.calls.Load())
	require.NotNil(t, service.GetModelPricing("remote-model"))
}

// 更新线程整体替换目录时，并发读者只看见一份完整价格，停止后目录仍可读取。
func TestPricingConcurrentReadAndReload(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.json")
	second := filepath.Join(dir, "second.json")
	require.NoError(t, os.WriteFile(first, []byte(`{"providers":{"openai":{"models":{"model":{"cost":{"input":1000000,"output":2000000}}}}}}`), 0o600))
	require.NoError(t, os.WriteFile(second, []byte(`{"providers":{"openai":{"models":{"model":{"cost":{"input":3000000,"output":4000000}}}}}}`), 0o600))
	service := NewService(Options{DataDir: dir}, nil)
	require.NoError(t, service.publishModelsCatalog(readCatalogTestFile(t, first), time.Now(), false))
	var readers sync.WaitGroup
	var inconsistent atomic.Bool
	for range 4 {
		readers.Go(func() {
			for range 200 {
				price := service.GetModelPricing("model")
				if price == nil || price.OutputCostPerToken-price.InputCostPerToken != 1 {
					inconsistent.Store(true)
				}
				_ = service.GetStatus()
				_ = service.Snapshot()
			}
		})
	}
	for i := range 20 {
		file := first
		if i%2 == 0 {
			file = second
		}
		require.NoError(t, service.publishModelsCatalog(readCatalogTestFile(t, file), time.Now(), false))
	}
	readers.Wait()
	require.False(t, inconsistent.Load())
	service.Stop()
	require.NotNil(t, service.GetModelPricing("model"))
}

// readCatalogTestFile 读取并发发布测试准备的目录正文。
func readCatalogTestFile(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	return body
}
