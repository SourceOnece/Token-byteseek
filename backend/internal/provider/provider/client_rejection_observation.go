package provider

import (
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/tidwall/gjson"
)

// ClientRejectionObservation 保留原精确关键词和 detail.code 分类。
func ClientRejectionObservation(message string, body []byte) provider.ClientRejectionObservation {
	lower := strings.ToLower(message)
	return provider.ClientRejectionObservation{Message: message, OrganizationDisabled: strings.Contains(lower, "organization has been disabled"), CreditBalanceExhausted: strings.Contains(lower, "credit balance"), IdentityVerificationRequired: strings.Contains(lower, "identity verification is required"), WorkspaceDeactivated: gjson.GetBytes(body, "detail.code").String() == "deactivated_workspace"}
}
