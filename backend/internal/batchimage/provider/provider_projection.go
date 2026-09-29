// 此处只投影现有提供商凭据能力与平台原语，不拥有提供商业务规则。
package provider

import (
	"context"
	"io"
	"strings"

	core "github.com/TokenFlux/TokenRouter/internal/batchimage"
	acct "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
)

type (
	Provider         = acct.Record
	GeminiTokenCache = acct.AccessTokenCache
)

const (
	PlatformGemini             = acct.PlatformGemini
	ProviderTypeAPIKey         = acct.ProviderTypeAPIKey
	ProviderTypeServiceAccount = acct.ProviderTypeServiceAccount
)

type BatchImageProvider interface {
	Name() string
	SupportsProvider(*Provider) bool
	Submit(context.Context, *core.BatchImageJob, *Provider, core.BatchImageInput) (*core.BatchProviderJob, error)
	Get(context.Context, *core.BatchImageJob, *Provider) (*core.BatchProviderStatus, error)
	Cancel(context.Context, *core.BatchImageJob, *Provider) error
	OpenResult(context.Context, *core.BatchImageJob, *Provider) (io.ReadCloser, string, error)
	Cleanup(context.Context, *core.BatchImageJob, *Provider, core.CleanupTarget) error
}

func batchImageProviderAPIKey(value *Provider) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(value.GetCredential("api_key"))
}

func resolveBatchProtocol(value *Provider) (capability.ProtocolID, bool) {
	plan := routing.Plan(routing.PlanInput{ClientProtocol: capability.ProtocolImageBatches})
	candidate, ok := plan.ResolveCandidate(value.RoutingSnapshot())
	return candidate.UpstreamProtocol, ok
}
