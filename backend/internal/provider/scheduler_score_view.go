package provider

// ProviderSchedulerScore 表示管理端展示的提供商调度评分。
type ProviderSchedulerScore struct {
	BaseScore             float64 `json:"base_score"`
	StickyScore           float64 `json:"sticky_score"`
	StickyScoreInfinity   bool    `json:"sticky_score_infinity"`
	StickyWeightedEnabled bool    `json:"sticky_weighted_enabled"`
}

// ProviderSchedulerGroupScore 表示提供商在指定分组中的调度评分。
type ProviderSchedulerGroupScore struct {
	GroupID   *int64 `json:"group_id"`
	GroupName string `json:"group_name,omitempty"`
	ProviderSchedulerScore
}
