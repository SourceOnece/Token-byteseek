package provider

import "time"

// BulkUpdate保留批量字段的显式零值与省略语义；nil表示不修改。
type BulkUpdate struct {
	Notes              *string
	ExpiresAt          *time.Time
	ClearExpiresAt     bool
	AutoPauseOnExpired *bool
	Name               *string
	ProxyID            *int64
	Concurrency        *int
	Priority           *int
	RateMultiplier     *float64
	LoadFactor         *int
	Status             *string
	Schedulable        *bool
	Credentials        map[string]any
	Extra              map[string]any
	// EnsureCodexFingerprintSeed 要求仓储原子保留或生成启用收敛的账号 seed。
	EnsureCodexFingerprintSeed bool
}
