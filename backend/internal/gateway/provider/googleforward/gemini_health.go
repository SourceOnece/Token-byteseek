package googleforward

import (
	"context"
	"net/http"

	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
)

// observeHealth 只回写原先会同步的凭据与附加状态，不把健康快照覆盖到执行提供商。
func (s *Gemini) observeHealth(ctx context.Context, target *gatewayadapter.ExecutionProvider, status int, headers http.Header, body []byte) {
	record := gatewayadapter.ExecutionRecord(target)
	updated := s.Errors.Observe(ctx, record, status, headers, body, gatewayadapter.HealthObservationFromContext(ctx, status, headers, body, nil))
	if target != nil && updated != nil {
		target.Record.Credentials, target.Record.Extra = updated.Credentials, updated.Extra
	}
}
