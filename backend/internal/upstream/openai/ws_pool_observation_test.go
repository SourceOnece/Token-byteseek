package openai

// ProviderPoolLoad 返回指定提供商连接池的并发与排队快照。
func (p *WSConnPool) ProviderPoolLoad(providerID int64) (inflight int, waiters int, conns int) {
	if p == nil || providerID <= 0 {
		return 0, 0, 0
	}
	ap, ok := p.getProviderPool(providerID)
	if !ok || ap == nil {
		return 0, 0, 0
	}
	ap.mu.Lock()
	defer ap.mu.Unlock()
	inflight, waiters = providerPoolLoadLocked(ap)
	return inflight, waiters, len(ap.conns)
}
