package selection

import (
	"context"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/requeststate"
)

func (s *Compatible) shadowProtocolsAllowed(ctx context.Context, provider *gatewayprovider.ExecutionProvider) bool {
	if provider == nil || !provider.View().IsShadow() {
		return true
	}
	if source, _ := requeststate.ClientProtocolFromContext(ctx); source == "" {
		return true
	}
	parent := s.parentProviderLookup(ctx)(*provider.Record.ParentProviderID)
	return parent != nil && gatewayprovider.ExecutionModelPolicy(parent).AllowsProtocol(ctx)
}
