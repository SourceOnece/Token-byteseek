package app

import (
	"time"

	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	"github.com/TokenFlux/TokenRouter/internal/batchimage"
	"github.com/TokenFlux/TokenRouter/internal/creative"

	batchhttp "github.com/TokenFlux/TokenRouter/internal/batchimage/httpapi"
	batchimageprovider "github.com/TokenFlux/TokenRouter/internal/batchimage/provider"
	"github.com/TokenFlux/TokenRouter/internal/config"

	creativehttp "github.com/TokenFlux/TokenRouter/internal/creative/httpapi"
)

func provideBatchRegistry(cfg *config.Config) *batchimage.Registry[batchimageprovider.BatchImageProvider] {
	return batchimage.NewRegistry[batchimageprovider.BatchImageProvider](batchimageprovider.NewGeminiAPIBatchImageProvider(nil), batchimageprovider.NewVertexBatchImageProvider(batchVertexOptions(cfg), nil, nil, nil))
}

func provideCreativeHTTP(s *creative.Public, activity *taskRequestActivity) *creativehttp.CreativeHandler {
	h := creativehttp.NewCreativeHandler(s)
	h.BindActivity(activity.Enter)
	return h
}

func provideBatchHTTP(s *batchimage.Public, d *batchimage.Download, c *batchimage.Cleanup, activity *taskRequestActivity) *batchhttp.BatchImageHandler {
	h := batchhttp.NewBatchImageHandler(s, d, c, batchImageAccessPorts())
	h.BindActivity(activity.Enter)
	return h
}

// provideBatchDownload 装配批量图片下载用例，复用提供商注册表。
func provideBatchDownload(repo batchimage.BatchImageRepository, providers *providerpostgres.ProviderStore, limiter batchimage.BatchImageDownloadLimiter, cfg *config.Config, registry *batchimage.Registry[batchimageprovider.BatchImageProvider]) *batchimage.Download {
	core := &batchimage.Download{Repo: repo, Limiter: limiter, ResolveProvider: (batchimageprovider.ResultAccess{Registry: registry, Providers: providers}).Download}
	if cfg != nil {
		core.Options = batchimage.DownloadOptions{MaxItems: cfg.BatchImage.MaxDownloadItemsZip, MaxBytes: cfg.BatchImage.MaxDownloadBytesPerRequest, Duration: time.Duration(cfg.BatchImage.MaxDownloadDurationSeconds) * time.Second}
	}
	return core
}

// provideBatchCleanup 注入批量图片清理配置和日志出口，运行循环由模块持有。
func provideBatchCleanup(repo batchimage.BatchImageRepository, providers *providerpostgres.ProviderStore, cfg *config.Config, registry *batchimage.Registry[batchimageprovider.BatchImageProvider]) *batchimage.Cleanup {
	core := &batchimage.Cleanup{Repo: repo, Now: time.Now, Observe: creativeObserve, ResolveProvider: (batchimageprovider.ResultAccess{Registry: registry, Providers: providers}).Cleanup}
	if cfg != nil {
		core.Options = batchimage.CleanupOptions{InputRetention: time.Duration(cfg.BatchImage.InputRetentionAfterTerminalHours) * time.Hour, Interval: time.Duration(cfg.BatchImage.CleanupIntervalMinutes) * time.Minute, BatchSize: cfg.BatchImage.CleanupBatchSize}
	}
	return core
}

// batchCleanupRuntime 区分清理循环与任务消费循环的生命周期实例。
type batchCleanupRuntime struct{ *batchimage.Runtime }

func provideBatchCleanupRuntime(core *batchimage.Cleanup, cfg *config.Config) *batchCleanupRuntime {
	enabled := core != nil && core.Repo != nil && cfg != nil && cfg.BatchImage.Enabled && core.CleanupInterval() > 0
	return &batchCleanupRuntime{batchimage.NewRuntime("batch image cleanup", enabled, core.Run)}
}

// taskRequestActivity 等待提交、下载及管理请求结束，再停止 task worker 和共享存储。
type taskRequestActivity struct{ *lifecycle.Operations }

func provideTaskActivity(manager *lifecycle.Manager) *taskRequestActivity {
	activity := &taskRequestActivity{lifecycle.NewOperations("TaskRequestsAndDownloads")}
	manager.Register(lifecycle.Hook{Name: "TaskRequestsAndDownloads", StopOrder: 16, Stop: activity.StopContext})
	return activity
}
