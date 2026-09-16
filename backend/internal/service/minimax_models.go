package service

// MiniMaxDefaultModelIDs 与已选上游模型预设对齐；不猜测价格、上下文或模态。
func MiniMaxDefaultModelIDs() []string {
	return []string{"MiniMax-M3", "MiniMax-M2.7", "MiniMax-M2.7-highspeed", "MiniMax-M2.5", "MiniMax-M2.5-highspeed", "MiniMax-M2.1", "MiniMax-M2.1-highspeed", "MiniMax-M2"}
}
