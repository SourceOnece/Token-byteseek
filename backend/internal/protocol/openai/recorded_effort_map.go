package openai

// GetOpenAIReasoningEffortFromReqBody 只提取请求中显式给出的档位。
// 显式值代表客户端真实请求，缺失字段时不从模型名称推导。
func GetOpenAIReasoningEffortFromReqBody(reqBody map[string]any) (value string, present bool) {
	if reqBody == nil {
		return "", false
	}

	// 优先读取 reasoning.effort。
	if reasoning, ok := reqBody["reasoning"].(map[string]any); ok {
		if effort, ok := reasoning["effort"].(string); ok {
			return NormalizeRecordedReasoningEffort(effort), true
		}
	}

	// 部分客户端通过顶层字段传递。
	if effort, ok := reqBody["reasoning_effort"].(string); ok {
		return NormalizeRecordedReasoningEffort(effort), true
	}

	return "", false
}
