package provider

// MiniMax 目录沿用本批已同步的 sub2api 预设，不凭名称推断价格或模态。
func MiniMaxDefaultModelIDs() []string {
	return []string{"MiniMax-M3", "MiniMax-M2.7", "MiniMax-M2.7-highspeed", "MiniMax-M2.5", "MiniMax-M2.5-highspeed", "MiniMax-M2.1", "MiniMax-M2.1-highspeed", "MiniMax-M2"}
}
