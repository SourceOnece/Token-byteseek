package openai_ws_v2

// SnapshotMetrics 返回当前 passthrough 指标快照。
func SnapshotMetrics() MetricsSnapshot {
	return MetricsSnapshot{
		UsageParseFailureTotal: passthroughUsageParseFailureTotal.Load(),
	}
}

// MetricsSnapshot 只投影实际记录的用量解析失败次数。
type MetricsSnapshot struct {
	UsageParseFailureTotal int64
}
