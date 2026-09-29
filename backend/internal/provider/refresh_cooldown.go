package provider

import (
	"time"
)

// RefreshCooldownVersion 将成功刷新与当时的临时停调绑定，避免清理新的管理员状态。
type RefreshCooldownVersion struct {
	CredentialVersion
	ParentProviderID *int64
	QuotaDimension   string
	Until            *time.Time
	Reason           string
}

func ObserveRefreshCooldown(value *Record) RefreshCooldownVersion {
	if value == nil {
		return RefreshCooldownVersion{}
	}
	return RefreshCooldownVersion{
		CredentialVersion: FailureVersion(value).CredentialVersion,
		ParentProviderID:  clonePointer(value.ParentProviderID), QuotaDimension: value.QuotaDimension,
		Until: clonePointer(value.TempUnschedulableUntil), Reason: value.TempUnschedulableReason,
	}
}
