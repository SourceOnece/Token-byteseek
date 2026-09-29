package bridge

// 独立追踪输出分片，终态补全文本时不能重复已经发送的 delta。
type responsesTextPart struct {
	OutputIndex  int
	ContentIndex int
}

// 显式关闭 thinking 优先于 output_config，沿用已同步 sub2api 的行为。
func anthropicReasoningEffort(req *AnthropicRequest) string {
	if req.Thinking != nil && req.Thinking.Type == "disabled" {
		return "none"
	}
	if req.OutputConfig != nil && req.OutputConfig.Effort != "" {
		return req.OutputConfig.Effort
	}
	return "medium"
}
