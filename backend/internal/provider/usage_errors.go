// 上游用量查询通过结构化错误区分配置、认证、限流及网络失败。
package provider

import (
	infraerrors "github.com/TokenFlux/TokenRouter/internal/pkg/apperror"
	"github.com/TokenFlux/TokenRouter/internal/upstream/usageview"
)

var (
	ErrUpstreamUsageUnavailable = infraerrors.ServiceUnavailable(
		"UPSTREAM_USAGE_UNAVAILABLE", "upstream usage query service is unavailable",
	)
	ErrUpstreamUsageProviderInvalid = infraerrors.BadRequest(
		"UPSTREAM_USAGE_PROVIDER_INVALID", "provider is not a supported API key provider",
	)
	ErrUpstreamUsageProviderDisabled = infraerrors.New(infraerrors.Category(422),
		"UPSTREAM_USAGE_PROVIDER_DISABLED", "provider is disabled",
	)
	ErrUpstreamUsageDisabled = infraerrors.New(infraerrors.Category(422),
		"UPSTREAM_USAGE_DISABLED", "upstream usage query is disabled for this provider",
	)
	ErrUpstreamUsageUnsupported = usageview.ErrUpstreamUsageUnsupported

	ErrUpstreamUsageWalletUnavailable = usageview.ErrUpstreamUsageWalletUnavailable

	ErrUpstreamUsageTimeout         = usageview.ErrUpstreamUsageTimeout
	ErrUpstreamUsageInvalidResponse = usageview.ErrUpstreamUsageInvalidResponse
	ErrUpstreamUsageRequestFailed   = usageview.ErrUpstreamUsageRequestFailed
	ErrUpstreamUsageIdentityChanged = infraerrors.Conflict(
		"UPSTREAM_USAGE_IDENTITY_CHANGED", "provider credentials or connection settings changed during the query",
	)
	ErrUpstreamUsageConfigInvalid = usageview.ErrUpstreamUsageConfigInvalid
	ErrUpstreamUsageBatchInvalid  = infraerrors.BadRequest(
		"UPSTREAM_USAGE_BATCH_INVALID", "upstream usage batch request is invalid",
	)
	ErrUpstreamUsageBatchTooLarge = infraerrors.BadRequest(
		"UPSTREAM_USAGE_BATCH_TOO_LARGE", "too many providers in one upstream usage query",
	)
)
