package service

import "context"

type codexQualityFilterKey struct{}

// WithCodexQualityFilter 仅管理列表使用，保持旧仓储接口及网关调度调用兼容。
func WithCodexQualityFilter(ctx context.Context, status string) context.Context {
	return context.WithValue(ctx, codexQualityFilterKey{}, status)
}
func CodexQualityFilter(ctx context.Context) string {
	value, _ := ctx.Value(codexQualityFilterKey{}).(string)
	return value
}
func ValidCodexQualityFilter(value string) bool {
	switch value {
	case "", "full", "degraded", "failed", "untested", "cancelled", "stale", "skipped":
		return true
	}
	return false
}
