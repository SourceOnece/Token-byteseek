package provider

import (
	"context"
	"errors"
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

func (r *catalogRemoteFixture) FetchPricingJSON(context.Context, string) ([]byte, error) {
	return r.body, r.err
}

func (r *catalogRemoteFixture) FetchHashText(context.Context, string) (string, error) {
	return "", errors.New("hash endpoint must not be called")
}

func (r *catalogRemoteFixture) FetchCatalog(_ context.Context, _ string, validator string) ([]byte, string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.validators = append(r.validators, validator)
	return r.body, r.etag, r.unchanged, r.err
}

func TestModelsCatalogAtomicUpdateAndUnpricedAttributes(t *testing.T) {
	remote := &catalogRemoteFixture{body: []byte(modelsCatalogFixture), etag: "v1"}
	s := NewPricingService(Options{ModelsDev: true, RemoteURL: "https://models.dev/catalog.json", DataDir: t.TempDir()}, remote)
	require.NoError(t, s.ForceUpdate())
	require.Equal(t, "No price", *s.ModelAttributes("attributes-only").DisplayName)
	require.False(t, *s.ModelAttributes("attributes-only").Temperature)
	price := s.GetModelPricing("claude-test")
	require.InDelta(t, 6e-6, price.CacheCreationInputTokenCostAbove1hr, 1e-12)
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
	require.NoError(t, s.SyncWithRemote())
	require.Equal(t, "v1", remote.validators[len(remote.validators)-1])
	require.Empty(t, s.AttributesSnapshot().LastError)
}

// 属性刷新必须保持旧价、身份回退和只读计费快照，不把展示目录变成新的收费门禁。
func TestLegacyPricingUnaffectedByAttributeRefresh(t *testing.T) {
	remote := &catalogRemoteFixture{body: []byte(modelsCatalogFixture)}
	s := NewPricingService(Options{DataDir: t.TempDir()}, remote)
	s.pricingData = map[string]*LiteLLMModelPricing{"claude-test": {InputCostPerToken: .000009, OutputCostPerToken: .000017}}
	before := s.GetModelPricing("claude-test")
	require.NoError(t, s.UpdateAttributes())
	require.Equal(t, before, s.GetModelPricing("claude-test"))
	require.Equal(t, before, s.ReadOnlySnapshot().GetModelPricing("claude-test"))
	require.NotEmpty(t, s.AttributesSnapshot().Version)
	remote.body = []byte(`{"providers":{}}`)
	version := s.AttributesSnapshot().Version
	require.Error(t, s.UpdateAttributes())
	require.Equal(t, version, s.AttributesSnapshot().Version)
	require.Equal(t, before, s.GetModelPricing("claude-test"))
}

func TestModelsCatalogOverrideAndConcurrentReaders(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "override.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"claude-test":{"cache_creation_input_token_cost_above_1hr":0.000012}}`), 0o600))
	remote := &catalogRemoteFixture{body: []byte(modelsCatalogFixture)}
	s := NewPricingService(Options{ModelsDev: true, RemoteURL: "https://models.dev/catalog.json", DataDir: dir, OverrideFile: file}, remote)
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
	s := NewPricingService(Options{ModelsDev: true, DataDir: t.TempDir()}, nil)
	require.NoError(t, s.Initialize())
	require.NotNil(t, s.ModelAttributes("claude-sonnet-4-5").Context)
	require.FileExists(t, s.GetPricingFilePath())
}

func TestModelsCatalogBrokenOverrideBootAndRecovery(t *testing.T) {
	dir := t.TempDir()
	patch := filepath.Join(dir, "override.json")
	require.NoError(t, os.WriteFile(patch, []byte("broken"), 0o600))
	s := NewPricingService(Options{ModelsDev: true, DataDir: dir, OverrideFile: patch}, nil)
	require.NoError(t, s.Initialize())
	require.NotEmpty(t, s.AttributesSnapshot().LastError)
	require.NotNil(t, s.ModelAttributes("claude-sonnet-4-5").Context)
	require.NoError(t, os.WriteFile(patch, []byte(`{"claude-sonnet-4-5":{"input_cost_per_token":0.000007}}`), 0o600))
	require.NoError(t, s.ReloadCustomPricingLayers())
	require.Empty(t, s.AttributesSnapshot().LastError)
	require.InDelta(t, 7e-6, s.GetModelPricing("claude-sonnet-4-5").InputCostPerToken, 1e-12)
}

func TestModelsCatalogReadOnlyCacheStillServesOfflineData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(path, []byte("file"), 0o600))
	s := NewPricingService(Options{ModelsDev: true, DataDir: path}, nil)
	require.NoError(t, s.Initialize())
	require.NotNil(t, s.ModelAttributes("claude-sonnet-4-5").Context)
}
