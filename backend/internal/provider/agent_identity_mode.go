// Agent Identity 模式由提供商的 OpenAI 认证配置决定。
package provider

import (
	"strings"
)

const OpenAIAuthModeAgentIdentity = "agentIdentity"

func (a *Record) IsOpenAIAgentIdentity() bool {
	if a == nil || !a.IsOpenAIOAuth() {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(a.GetCredential(OpenAIAuthModeCredentialKey)), OpenAIAuthModeAgentIdentity)
}
