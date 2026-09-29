package provider

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// UsageObservationVersion 限定上游观测可以写入的提供商身份与原健康窗口，不进入公开输出。
type UsageObservationVersion struct {
	CredentialVersion
	ParentProviderID                               *int64
	QuotaDimension                                 string
	RateLimitedAt, RateLimitResetAt, OverloadUntil *time.Time
}

// UsageObservationWriter 保留快照、设置限流、清除限流各自的提交与通知边界。
type UsageExtraWriter interface {
	UpdateUsageExtraIfUnchanged(context.Context, UsageObservationVersion, map[string]any) (bool, error)
}
type UsageObservationWriter interface {
	UsageExtraWriter
	SetUsageRateLimitIfUnchanged(context.Context, UsageObservationVersion, time.Time) (bool, error)
	ClearUsageRateLimitIfUnchanged(context.Context, UsageObservationVersion) (bool, error)
}

var (
	ErrUsageObservationChanged       = errors.New("provider identity changed during usage query")
	ErrUsageObservationWriterMissing = errors.New("usage observation conditional writer is not configured")
)

func ObserveUsageVersion(value *Record) UsageObservationVersion {
	if value == nil {
		return UsageObservationVersion{}
	}
	return UsageObservationVersion{
		CredentialVersion: FailureVersion(value).CredentialVersion,
		ParentProviderID:  clonePointer(value.ParentProviderID), QuotaDimension: value.QuotaDimension,
		RateLimitedAt: clonePointer(value.RateLimitedAt), RateLimitResetAt: clonePointer(value.RateLimitResetAt), OverloadUntil: clonePointer(value.OverloadUntil),
	}
}

// UsageCacheIdentity 仅在进程内校验缓存来源，不改变 Redis 协议、提供商 key 或 TTL。
func UsageCacheIdentity(value *Record) string {
	if value == nil {
		return ""
	}
	parent := int64(0)
	if value.ParentProviderID != nil {
		parent = *value.ParentProviderID
	}
	return fmt.Sprintf("%s/%s/%d/%s", RefreshCredentialIdentity(value), value.Status, parent, value.QuotaDimension)
}
