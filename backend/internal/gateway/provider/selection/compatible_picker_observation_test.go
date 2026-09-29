package selection

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
)

// isProviderRequestCompatible 为布尔断言投影实际兼容性判定。
func (s *compatiblePicker) isProviderRequestCompatible(ctx context.Context, provider *provider.ExecutionProvider, req scheduler.PlatformSelectionInput) bool {
	compatible, _ := s.isProviderRequestCompatibleReason(ctx, provider, req)
	return compatible
}
