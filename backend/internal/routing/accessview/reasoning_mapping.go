// 推理强度映射同时记录来源、目标以及可选的模型匹配范围。
package accessview

type ReasoningEffortMapping struct {
	From      string `json:"from"`
	To        string `json:"to"`
	MatchType string `json:"match_type,omitempty"`
	Model     string `json:"model,omitempty"`
}
