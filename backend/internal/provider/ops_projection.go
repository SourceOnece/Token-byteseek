// LoadObservation 只投影观测所需提供商 ID 与有效负载上限。
package provider

type LoadObservation struct {
	ID             int64
	MaxConcurrency int
}
