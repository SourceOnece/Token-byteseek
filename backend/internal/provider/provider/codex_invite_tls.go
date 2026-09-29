package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/egress"
	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient/tlsfingerprint"
	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// resolveCodexInviteTLS 保留专用 Router 优先、提供商直接配置回退的选择次序。
func resolveCodexInviteTLS(profiles *egressprovider.TLSProfiles, value *provider.Record, router *egress.TLSFingerprintRouter) *tlsfingerprint.Profile {
	if profiles == nil {
		return nil
	}
	if router != nil && router.CodexInviteResetTLSFingerprintProfileID != nil {
		if profile, ok := profiles.ResolveTokenTLSProfileByID(*router.CodexInviteResetTLSFingerprintProfileID); ok {
			return profile
		}
	}
	return profiles.ResolveRequestTLS(egress.TLSSelection{
		Enabled: value.IsTLSFingerprintEnabled(), DirectProfileID: value.GetTLSFingerprintProfileID(),
	})
}
