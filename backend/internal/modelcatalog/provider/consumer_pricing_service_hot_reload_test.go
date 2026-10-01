package provider

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/billing/pricing"

	"github.com/stretchr/testify/require"
)

const hotReloadCatalogJSON = `{"providers":{"openai":{"models":{"remote-model":{"cost":{"input":1,"output":2}}}}}}`

func hotReloadModelJSON(name string, input, output float64) string {
	return `"` + name + `": {"provider": "test", "mode": "chat",
		"input_cost_per_token": ` + formatFloat(input) + `, "output_cost_per_token": ` + formatFloat(output) + `}`
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// newHotReloadCatalog 在数据目录里放好目录缓存与补充文件并完成首次加载。
// 传空串表示不配置补充文件。
func newHotReloadCatalog(t *testing.T, fallbackJSON string) *Service {
	t.Helper()
	dir := t.TempDir()
	svc := newModelCatalogFixture(modelCatalogFixture{options: Options{}})
	svc.options.DataDir = dir
	require.NoError(t, os.WriteFile(svc.catalogFilePath(), []byte(hotReloadCatalogJSON), 0o644))
	if fallbackJSON != "" {
		svc.options.FallbackFile = filepath.Join(dir, "fallback.json")
		require.NoError(t, os.WriteFile(svc.options.FallbackFile, []byte(fallbackJSON), 0o644))
	}
	require.NoError(t, svc.Initialize())
	return svc
}

func TestPricingHotReload_FallbackChangeRebuildsWithoutTouchingSyncAnchor(t *testing.T) {
	svc := newHotReloadCatalog(t, `{`+hotReloadModelJSON("custom-a", 4e-6, 8e-6)+`}`)
	require.InDelta(t, 4e-6, svc.Snapshot().Data["custom-a"].InputCostPerToken, 1e-12)
	require.Nil(t, svc.Snapshot().Data["custom-b"])
	require.NotEmpty(t, svc.Snapshot().CustomFilesHash)
	anchor, updated := svc.Snapshot().LocalHash, svc.Snapshot().LastUpdated

	require.NoError(t, os.WriteFile(svc.options.FallbackFile, []byte(`{`+
		hotReloadModelJSON("custom-a", 5e-6, 8e-6)+`,`+
		hotReloadModelJSON("custom-b", 1e-6, 3e-6)+`}`), 0o644))
	svc.reloadIfCustomFilesChanged()

	require.InDelta(t, 5e-6, svc.Snapshot().Data["custom-a"].InputCostPerToken, 1e-12, "改价即时生效")
	require.NotNil(t, svc.Snapshot().Data["custom-b"], "新模型即时并入")
	require.InDelta(t, 1e-6, svc.Snapshot().Data["remote-model"].InputCostPerToken, 1e-12, "目录条目不受影响")
	require.Equal(t, anchor, svc.Snapshot().LocalHash, "热重载不得改动远程同步锚点")
	require.Equal(t, updated, svc.Snapshot().LastUpdated)
	require.Equal(t, svc.customPricingFilesFingerprint(), svc.Snapshot().CustomFilesHash)
}

func TestPricingHotReload_InvalidFileKeepsCurrentDataUntilFixed(t *testing.T) {
	svc := newHotReloadCatalog(t, `{`+hotReloadModelJSON("custom-a", 4e-6, 8e-6)+`}`)
	before := svc.Snapshot().CustomFilesHash

	require.NoError(t, os.WriteFile(svc.options.FallbackFile, []byte(`{"custom-a": {"input_cost_per_token": `), 0o644))
	svc.reloadIfCustomFilesChanged()
	require.InDelta(t, 4e-6, svc.Snapshot().Data["custom-a"].InputCostPerToken, 1e-12, "半写文件不得替换数据")
	require.Equal(t, before, svc.Snapshot().CustomFilesHash, "指纹不更新，下一轮继续尝试")

	require.NoError(t, os.WriteFile(svc.options.FallbackFile, []byte(`{`+hotReloadModelJSON("custom-a", 6e-6, 8e-6)+`}`), 0o644))
	svc.reloadIfCustomFilesChanged()
	require.InDelta(t, 6e-6, svc.Snapshot().Data["custom-a"].InputCostPerToken, 1e-12, "文件修好后正常重建")
	require.NotEqual(t, before, svc.Snapshot().CustomFilesHash)
}

type stubCatalogRemoteClient struct{ body string }

func (c stubCatalogRemoteClient) FetchCatalog(context.Context, string, string) ([]byte, string, bool, error) {
	return []byte(c.body), "", false, nil
}

// 远程下载重建后指纹必须同步到当前文件内容，否则下一轮定时比对会多做一次无意义重载。
func TestPricingHotReload_DownloadRefreshesFingerprint(t *testing.T) {
	svc := newHotReloadCatalog(t, `{`+hotReloadModelJSON("custom-a", 4e-6, 8e-6)+`}`)
	svc.options.RemoteURL = "https://example.com/pricing.json"
	setPricingFixtureRemote(svc, stubCatalogRemoteClient{body: hotReloadCatalogJSON})
	require.NoError(t, os.WriteFile(svc.options.FallbackFile, []byte(`{`+hotReloadModelJSON("custom-b", 1e-6, 3e-6)+`}`), 0o644))

	require.NoError(t, svc.ForceUpdate())

	require.NotNil(t, svc.Snapshot().Data["custom-b"])
	require.Nil(t, svc.Snapshot().Data["custom-a"])
	require.Equal(t, svc.customPricingFilesFingerprint(), svc.Snapshot().CustomFilesHash)

	mutatePricingFixture(svc, func(data map[string]*pricing.CatalogModelPricing) {
		data["sentinel"] = &pricing.CatalogModelPricing{}
	})
	svc.reloadIfCustomFilesChanged()
	require.Contains(t, svc.Snapshot().Data, "sentinel", "下载已消化文件变化，不得再次重建")
}

// TestPricingSchedulerStartsForCustomFilesWithoutRemoteURL 验证只配置了 补充文件 而没有 remote_url 时调度器也要运行，否则文件改动无人比对。
func TestPricingSchedulerStartsForCustomFilesWithoutRemoteURL(t *testing.T) {
	svc := NewService(Options{
		RemoteURL:    "",
		FallbackFile: filepath.Join(t.TempDir(), "fallback.json"),
	}, nil)

	svc.startUpdateScheduler()
	done := make(chan struct{})
	go func() {
		svc.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("custom file watch must keep the scheduler running")
	case <-time.After(50 * time.Millisecond):
	}

	svc.Stop()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler must exit after Stop")
	}
}

// TestSupplementRemovalAndRecreation 更新、删除和重新创建补充文件时保留原目录及条件请求锚点。
func TestSupplementRemovalAndRecreation(t *testing.T) {
	svc := newHotReloadCatalog(t, `{"custom-model":{"input_cost_per_token":0.000004,"output_cost_per_token":0.000008}}`)
	before := svc.Snapshot()
	require.NoError(t, os.WriteFile(svc.options.FallbackFile, []byte(`{"custom-model":{"input_cost_per_token":0.000007,"output_cost_per_token":0.000008}}`), 0o600))
	require.NoError(t, svc.ForceUpdate())
	require.InDelta(t, 7e-6, svc.GetModelPricing("custom-model").InputCostPerToken, 1e-12)
	require.NoError(t, os.Remove(svc.options.FallbackFile))
	svc.reloadIfCustomFilesChanged()
	require.Nil(t, svc.GetModelPricing("custom-model"))
	require.Equal(t, before.LocalHash, svc.Snapshot().LocalHash)
	require.Equal(t, before.LastUpdated, svc.Snapshot().LastUpdated)
	hash := svc.Snapshot().CustomFilesHash
	svc.reloadIfCustomFilesChanged()
	require.Equal(t, hash, svc.Snapshot().CustomFilesHash)
	require.NoError(t, os.WriteFile(svc.options.FallbackFile, []byte(`{"custom-model":{"input_cost_per_token":0,"output_cost_per_token":0}}`), 0o600))
	svc.reloadIfCustomFilesChanged()
	require.NotNil(t, svc.GetModelPricing("custom-model"))
	require.Zero(t, svc.GetModelPricing("custom-model").InputCostPerToken)
}
