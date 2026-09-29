package httpapi

import (
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
)

func activeCodexFingerprintMode(provider *gatewayprovider.ExecutionProvider) providercore.CodexFingerprintMode {
	if provider == nil || gatewayprovider.ExecutionProtocolRecord(provider).GetCodexFingerprintMode() == providercore.CodexFingerprintOff {
		return providercore.CodexFingerprintOff
	}
	if _, ok := providercore.CodexFingerprintSeed(provider.Record.Extra); !ok {
		return providercore.CodexFingerprintOff
	}
	return gatewayprovider.ExecutionProtocolRecord(provider).GetCodexFingerprintMode()
}

func openAIWSPoolProviderView(provider *gatewayprovider.ExecutionProvider) *openai.WSPoolProvider {
	if provider == nil {
		return nil
	}
	return &openai.WSPoolProvider{ID: provider.Record.ID, Concurrency: provider.Record.Concurrency, Type: provider.Record.Type, FingerprintMode: string(activeCodexFingerprintMode(provider))}
}
