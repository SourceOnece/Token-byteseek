package provider

// ProviderBulkUpdate 表达批量配置补丁；nil 指针表示不修改对应字段。
type ProviderBulkUpdate struct {
	// ProtocolUpdates 是校验后的逐提供商非敏感协议补丁，同一 SQL 原子合并。
	ProtocolUpdates map[int64]map[string]any

	Name           *string
	ProxyID        *int64
	Concurrency    *int
	Priority       *int
	RateMultiplier *float64
	LoadFactor     *int
	Status         *string
	Schedulable    *bool
	Credentials    map[string]any
	Extra          map[string]any
	// EnsureCodexFingerprintSeed 要求仓储原子保留或生成启用收敛的提供商 seed。
	EnsureCodexFingerprintSeed bool
}
