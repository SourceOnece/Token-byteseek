package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/pkg/apperror"
)

var (
	ErrProviderNotFound      = billing.ErrProviderNotFound
	ErrProviderNilInput      = apperror.BadRequest("PROVIDER_NIL_INPUT", "provider input cannot be nil")
	ErrProviderNotInFallback = apperror.BadRequest("PROVIDER_NOT_IN_FALLBACK", "provider is not in proxy fallback state")
)

// DiscardDeprecatedExtra 保持原写入边界的废弃键清理。
func DiscardDeprecatedExtra(extra map[string]any) {
	delete(extra, "upstream_billing_probe")
	delete(extra, "upstream_billing_probe_enabled")
	delete(extra, "openai_long_context_billing_enabled")
}
