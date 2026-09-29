package provider

import (
	"context"
	"net/http"

	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// ApplyExecutionSchedulingThreshold 只回写本次健康裁决明确拥有的字段，保留执行快照隔离。
func ApplyExecutionSchedulingThreshold(ctx context.Context, observer *provideradapter.UpstreamHealth, value *ExecutionProvider) bool {
	if observer == nil {
		return false
	}
	record := ExecutionRecord(value)
	paused := observer.Core.ApplyProviderSchedulingThreshold(ctx, record)
	if value != nil && record != nil {
		value.Record.TempUnschedulableUntil = record.TempUnschedulableUntil
		value.Record.TempUnschedulableReason = record.TempUnschedulableReason
		value.Record.Extra = record.Extra
	}
	return paused
}

// ObserveExecutionSessionWindow 在空观测时先短路，不触碰可选健康依赖。
func ObserveExecutionSessionWindow(ctx context.Context, observer *provideradapter.UpstreamHealth, value *ExecutionProvider, headers http.Header) {
	observation := provideradapter.SessionWindowObservation(headers)
	if observation.Status == "" {
		return
	}
	observer.Core.UpdateSessionWindow(ctx, ExecutionRecord(value), observation)
}
