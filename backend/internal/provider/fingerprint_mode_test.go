package provider_test

import (
	"testing"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/assert"
)

func newTestOAuthProvider(id int64, extra map[string]any) *providercore.Record {
	if providercore.CodexFingerprintModeRequiresSeed(providercore.CodexFingerprintModeFromExtra(extra)) {
		if extra == nil {
			extra = make(map[string]any)
		}
		if _, exists := extra[providercore.CodexFingerprintSeedExtraKey]; !exists {
			extra[providercore.CodexFingerprintSeedExtraKey] = testCodexFingerprintSeed
		}
	}
	return &providercore.Record{
		ID:       id,
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeOAuth,
		Extra:    extra,
	}
}

func TestGetCodexFingerprintMode(t *testing.T) {
	tests := []struct {
		name     string
		provider *providercore.Record
		expected providercore.CodexFingerprintMode
	}{
		{"nil 提供商", nil, providercore.CodexFingerprintOff},
		{"非 OAuth 提供商", &providercore.Record{Platform: capability.PlatformOpenAI, Type: "api_key"}, providercore.CodexFingerprintOff},
		{"OpenAI setup token", &providercore.Record{Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeSetupToken, Extra: map[string]any{providercore.CodexFingerprintModeExtraKey: "session"}}, providercore.CodexFingerprintSession},
		{"Anthropic setup token", &providercore.Record{Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeSetupToken, Extra: map[string]any{providercore.CodexFingerprintModeExtraKey: "session"}}, providercore.CodexFingerprintOff},
		// 收敛是显式 opt-in：缺省/空/非法一律 off（#5610）。存量提供商普遍没有这个
		// extra 键，升级不得把它们静默切进收敛。
		{"无 extra 默认 off", newTestOAuthProvider(1, nil), providercore.CodexFingerprintOff},
		{"空值默认 off", newTestOAuthProvider(1, map[string]any{providercore.CodexFingerprintModeExtraKey: ""}), providercore.CodexFingerprintOff},
		{"非法值默认 off", newTestOAuthProvider(1, map[string]any{providercore.CodexFingerprintModeExtraKey: "invalid"}), providercore.CodexFingerprintOff},
		{"显式 off", newTestOAuthProvider(1, map[string]any{providercore.CodexFingerprintModeExtraKey: "off"}), providercore.CodexFingerprintOff},
		{"device", newTestOAuthProvider(1, map[string]any{providercore.CodexFingerprintModeExtraKey: "device"}), providercore.CodexFingerprintDevice},
		{"session", newTestOAuthProvider(1, map[string]any{providercore.CodexFingerprintModeExtraKey: "session"}), providercore.CodexFingerprintSession},
		{"full", newTestOAuthProvider(1, map[string]any{providercore.CodexFingerprintModeExtraKey: "full"}), providercore.CodexFingerprintFull},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.provider.GetCodexFingerprintMode())
		})
	}
}

const testCodexFingerprintSeed = "11111111-1111-4111-8111-111111111111"
