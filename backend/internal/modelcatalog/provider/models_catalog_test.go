package provider

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

const modelsCatalogFixture = `{"models":{"anthropic/claude-test":{"name":"Claude"}},"providers":{"anthropic":{"models":{"claude-test":{"name":"Claude","reasoning":true,"limit":{"output":10},"cost":{"input":3,"output":15,"cache_write":3.75,"cache_read":0.3,"tiers":[{"tier":{"type":"context","size":100},"input":6,"output":30,"cache_write":7.5,"cache_read":0.6},{"tier":{"type":"context","size":200},"input":9,"output":45,"cache_write":11.25,"cache_read":0.9}]}},"attributes-only":{"name":"No price","temperature":false}}}}}`

type catalogRemoteFixture struct {
	mu         sync.Mutex
	body       []byte
	err        error
	etag       string
	unchanged  bool
	validators []string
}

func (r *catalogRemoteFixture) FetchCatalog(_ context.Context, _ string, validator string) ([]byte, string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.validators = append(r.validators, validator)
	return r.body, r.etag, r.unchanged, r.err
}

func TestModelsCatalogAtomicUpdateAndUnpricedAttributes(t *testing.T) {
	remote := &catalogRemoteFixture{body: []byte(modelsCatalogFixture), etag: "v1"}
	s := NewService(Options{RemoteURL: "https://models.dev/catalog.json", DataDir: t.TempDir()}, remote)
	require.NoError(t, s.ForceUpdate())
	require.Equal(t, "No price", *s.ModelAttributes("attributes-only").DisplayName)
	require.False(t, *s.ModelAttributes("attributes-only").Temperature)
	price := s.GetModelPricing("claude-test")
	require.False(t, price.CacheCreation1hPricePresent)
	require.Zero(t, price.CacheCreationInputTokenCostAbove1hr)
	require.Len(t, price.ContextPrices, 2)
	before := s.AttributesSnapshot()
	remote.body = []byte(`{"providers":{}}`)
	require.Error(t, s.ForceUpdate())
	after := s.AttributesSnapshot()
	require.Equal(t, before.Version, after.Version)
	require.Equal(t, before.LastUpdated, after.LastUpdated)
	require.NotEmpty(t, after.LastError)
	require.Equal(t, price.InputCostPerToken, s.GetModelPricing("claude-test").InputCostPerToken)
	remote.unchanged = true
	require.NoError(t, s.syncWithRemote())
	require.Equal(t, "v1", remote.validators[len(remote.validators)-1])
	require.Empty(t, s.AttributesSnapshot().LastError)
}

func TestModelsCatalogSupplementAndConcurrentReaders(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "supplement.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"claude-test":{"cache_creation_input_token_cost_above_1hr":0.000012}}`), 0o600))
	remote := &catalogRemoteFixture{body: []byte(modelsCatalogFixture)}
	s := NewService(Options{RemoteURL: "https://models.dev/catalog.json", DataDir: dir, FallbackFile: file}, remote)
	require.NoError(t, s.ForceUpdate())
	require.InDelta(t, 12e-6, s.GetModelPricing("claude-test").CacheCreationInputTokenCostAbove1hr, 1e-12)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				_ = s.ModelAttributes("claude-test")
				_ = s.AttributesSnapshot()
				_ = s.ReadOnlySnapshot()
			}
		}()
	}
	for range 3 {
		require.NoError(t, s.ForceUpdate())
	}
	wg.Wait()
}

func TestModelsCatalogOfflineFirstStart(t *testing.T) {
	s := NewService(Options{DataDir: t.TempDir()}, nil)
	require.NoError(t, s.Initialize())
	require.NotNil(t, s.ModelAttributes("claude-sonnet-4-5").Context)
	require.FileExists(t, s.catalogFilePath())
}

func TestModelsCatalogBrokenSupplementBootAndRecovery(t *testing.T) {
	dir := t.TempDir()
	patch := filepath.Join(dir, "supplement.json")
	require.NoError(t, os.WriteFile(patch, []byte("broken"), 0o600))
	s := NewService(Options{DataDir: dir, FallbackFile: patch}, nil)
	require.NoError(t, s.Initialize())
	require.NotEmpty(t, s.AttributesSnapshot().LastError)
	require.NotNil(t, s.ModelAttributes("claude-sonnet-4-5").Context)
	require.Nil(t, s.GetModelPricing("claude-opus-4-6-thinking"))
	require.NoError(t, os.WriteFile(patch, []byte(`{"custom-model":{"input_cost_per_token":0.000007}}`), 0o600))
	require.NoError(t, s.reloadCustomPricingLayers())
	require.Empty(t, s.AttributesSnapshot().LastError)
	require.InDelta(t, 7e-6, s.GetModelPricing("custom-model").InputCostPerToken, 1e-12)
}

func TestModelsCatalogReadOnlyCacheStillServesOfflineData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(path, []byte("file"), 0o600))
	s := NewService(Options{DataDir: path}, nil)
	require.NoError(t, s.Initialize())
	require.NotNil(t, s.ModelAttributes("claude-sonnet-4-5").Context)
}

// TestModelsCatalogConditionalUpdate 验证普通更新使用 ETag，强制更新清空条件，304 仍应用本地补充。
func TestModelsCatalogConditionalUpdate(t *testing.T) {
	dir := t.TempDir()
	patch := filepath.Join(dir, "supplement.json")
	remote := &catalogRemoteFixture{body: []byte(modelsCatalogFixture), etag: "v1"}
	service := NewService(Options{RemoteURL: "https://models.dev/catalog.json", DataDir: dir, FallbackFile: patch}, remote)
	require.NoError(t, service.ForceUpdate())
	require.NoError(t, service.syncWithRemote())
	require.Equal(t, []string{"", "v1"}, remote.validators)
	require.NoError(t, service.ForceUpdate())
	require.Equal(t, "", remote.validators[2])
	before := service.Snapshot()
	remote.unchanged = true
	require.NoError(t, os.WriteFile(patch, []byte(`{"custom-model":{"input_cost_per_token":0}}`), 0o600))
	require.NoError(t, service.syncWithRemote())
	require.Zero(t, service.GetModelPricing("custom-model").InputCostPerToken)
	require.Equal(t, before.LocalHash, service.Snapshot().LocalHash)
	require.Equal(t, before.LastUpdated, service.Snapshot().LastUpdated)
	require.NoError(t, os.Remove(patch))
	require.NoError(t, service.syncWithRemote())
	require.Nil(t, service.GetModelPricing("custom-model"))
}

// TestModelsCatalogRejectsLegacyRemote 验证自定义旧远程文件报迁移错误，内存和磁盘版本均保持不变。
func TestModelsCatalogRejectsLegacyRemote(t *testing.T) {
	remote := &catalogRemoteFixture{body: []byte(modelsCatalogFixture)}
	service := NewService(Options{RemoteURL: "https://custom.example/prices.json", DataDir: t.TempDir()}, remote)
	require.NoError(t, service.ForceUpdate())
	before := service.Snapshot()
	attrs := service.AttributesSnapshot()
	disk := readCatalogTestFile(t, service.catalogFilePath())
	remote.body = []byte(`{"gpt-test":{"input_cost_per_token":1}}`)
	require.ErrorContains(t, service.ForceUpdate(), "legacy pricing JSON")
	require.Equal(t, before.Data, service.Snapshot().Data)
	require.Equal(t, attrs.Items, service.AttributesSnapshot().Items)
	require.Equal(t, before.LocalHash, service.Snapshot().LocalHash)
	require.Equal(t, before.LastUpdated, service.Snapshot().LastUpdated)
	require.Equal(t, disk, readCatalogTestFile(t, service.catalogFilePath()))
}

// TestModelsCatalogDroppedAbsoluteTierWarns 验证绝对阶梯消失时沿用更新告警。
func TestModelsCatalogDroppedAbsoluteTierWarns(t *testing.T) {
	sink, restore := captureStructuredLog(t)
	defer restore()
	remote := &catalogRemoteFixture{body: []byte(modelsCatalogFixture)}
	service := NewService(Options{RemoteURL: "https://models.dev/catalog.json", DataDir: t.TempDir()}, remote)
	require.NoError(t, service.ForceUpdate())
	remote.body = []byte(`{"providers":{"anthropic":{"models":{"claude-test":{"cost":{"input":3,"output":15}}}}}}`)
	require.NoError(t, service.ForceUpdate())
	require.True(t, sink.ContainsMessageAtLevel("Long-context ladder dropped", "warn"))
}

// TestModelsCatalogAttributesOnlyWithUnknownPatch 验证没有价格的合法目录仍能发布属性，无价格补充不生成价格。
func TestModelsCatalogAttributesOnlyWithUnknownPatch(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "supplement.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"typo":{"long_context_input_token_threshold":0}}`), 0o600))
	remote := &catalogRemoteFixture{body: []byte(`{"providers":{"openai":{"models":{"attributes-only":{"name":"No prices"}}}}}`)}
	service := NewService(Options{RemoteURL: "https://models.dev/catalog.json", DataDir: dir, FallbackFile: file}, remote)
	require.NoError(t, service.ForceUpdate())
	require.NotContains(t, service.Snapshot().Data, "typo")
	require.Nil(t, service.GetModelPricing("attributes-only"))
	require.Equal(t, "No prices", *service.ModelAttributes("attributes-only").DisplayName)
}
