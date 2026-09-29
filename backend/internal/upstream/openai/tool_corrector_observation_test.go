package openai

// ResetStats 重置统计信息
func (c *CodexToolCorrector) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.TotalCorrected = 0
	c.stats.CorrectionsByTool = make(map[string]int)
}

// GetToolNameMapping 获取工具名称映射表
func GetToolNameMapping() map[string]string {
	// 返回副本以避免外部修改
	mapping := make(map[string]string, len(codexToolNameMapping))
	for k, v := range codexToolNameMapping {
		mapping[k] = v
	}
	return mapping
}
