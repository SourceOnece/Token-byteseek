package accessview

// GroupModelAllowlist 只按客户端原模型名准入，与价卡和展示列表独立。
type GroupModelAllowlist struct {
	Enabled bool     `json:"enabled"`
	Models  []string `json:"models,omitempty"`
}

func CloneModelAllowlist(v GroupModelAllowlist) GroupModelAllowlist {
	v.Models = append([]string(nil), v.Models...)
	return v
}
