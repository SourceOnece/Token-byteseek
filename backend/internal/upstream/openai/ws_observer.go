package openai

import "net/http"

// WSHandshakeObserver 属于物理握手；连接复用不替换观察身份。
// 上层只提供窄端口，传输池不依赖票据存储或账号调度实现。
type WSHandshakeObserver interface {
	ForHeaders(http.Header) WSHandshakeObserver
	ObserveHeader(http.Header)
	ObserveJSON([]byte, string, string)
}

// ObserveTicket 使用当前连接的收据和本轮实际出站模型，避免跨模型误归因。
func (l *WSConnLease) ObserveTicket(raw []byte, model string) {
	if l != nil && l.Conn != nil && l.Conn.observer != nil {
		l.Conn.observer.ObserveJSON(raw, "", model)
	}
}
