package selection

import (
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
)

// providers 为诊断测试投影候选列表，保留 nil 列表语义。
func (s *diagnosticScope) providers(values []*provider.ExecutionProvider) []*scheduler.DiagnosticProvider {
	if values == nil {
		return nil
	}
	out := make([]*scheduler.DiagnosticProvider, len(values))
	for i, v := range values {
		out[i] = s.provider(v)
	}
	return out
}
