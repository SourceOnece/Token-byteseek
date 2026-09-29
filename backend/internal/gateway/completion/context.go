package completion

import (
	"context"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/gateway/modeltrace"
	"github.com/TokenFlux/TokenRouter/internal/infra/telemetry"
)

type responseModelContextKey struct{}

// WithResponseModel 固化管理员诊断值，不把 Gin 或可变观察器交给异步结算。
func WithResponseModel(ctx context.Context, model string) context.Context {
	return context.WithValue(ctx, responseModelContextKey{}, strings.TrimSpace(model))
}

func ResponseModel(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(responseModelContextKey{}).(string)
	return value
}

// SnapshotContext 只固化原有三个关联字段及模型链，不继承请求取消或 Gin。
func SnapshotContext(source context.Context) context.Context {
	return copyCompletionContext(context.Background(), source)
}

func copyCompletionContext(destination, source context.Context) context.Context {
	if destination == nil {
		destination = context.Background()
	}
	if source == nil {
		return destination
	}
	destination = WithResponseModel(destination, ResponseModel(source))
	for _, key := range []telemetry.ContextKey{telemetry.RequestID, telemetry.ClientRequestID, telemetry.ClientModel} {
		if value := source.Value(key); value != nil {
			destination = context.WithValue(destination, key, value)
		}
	}
	if trace, ok := modeltrace.FromContext(source); ok {
		snapshot := trace.Snapshot()
		destination = modeltrace.WithContext(destination, snapshot)
		if strings.TrimSpace(snapshot.ClientModel) != "" {
			destination = context.WithValue(destination, telemetry.ClientModel, snapshot.ClientModel)
		}
	}
	return destination
}

// WrapTaskContext 在提交时冻结来源，执行时保留 worker 自己的预算。
func WrapTaskContext(source context.Context, task func(context.Context)) func(context.Context) {
	if task == nil {
		return nil
	}
	snapshot := SnapshotContext(source)
	return func(ctx context.Context) { task(copyCompletionContext(ctx, snapshot)) }
}
