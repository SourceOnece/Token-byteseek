package provider

// size 在锁内读取暂存条目数，供缓存生命周期测试核对。
func (s *ModelTransientState) size() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}
