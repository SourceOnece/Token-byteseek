package provider

// 内部持久化键保持旧值，不能随模块命名迁移删除或重命名已部署配置。
const (
	OllamaCloudUsageSessionExtraKey     = "ollama_cloud_usage_session"
	OllamaCloudUsageAutoRefreshExtraKey = "ollama_cloud_usage_auto_refresh"
	OllamaCloudUsageSnapshotExtraKey    = "ollama_cloud_usage_snapshot"
	CNUsageMonitorSnapshotExtraKey      = "cn_usage_monitor_snapshot"
	UpstreamUsageQueryExtraKey          = "upstream_usage_query"
)
