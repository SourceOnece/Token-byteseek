package domain

// GroupModelAllowlist 是显式启用的调用准入规则，与旧模型展示列表独立。
type GroupModelAllowlist struct {
	Enabled bool     `json:"enabled"`
	Models  []string `json:"models,omitempty"`
}
