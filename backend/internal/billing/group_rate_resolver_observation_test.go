package billing

// GroupRateCacheStats 读取分组倍率缓存的累计观测值。
func GroupRateCacheStats() (int64, int64, int64, int64, int64) {
	return groupRateMetrics.Hit.Load(), groupRateMetrics.Miss.Load(), groupRateMetrics.Load.Load(), groupRateMetrics.Shared.Load(), groupRateMetrics.Fallback.Load()
}
