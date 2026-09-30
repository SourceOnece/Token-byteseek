package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 同步展示属性不能让存量部署悄悄换报价来源。
func TestPricingCatalogSourcePreservesExistingDeployment(t *testing.T) {
	for _, source := range []string{
		"https://raw.githubusercontent.com/Wei-Shaw/model-price-repo/main/model_prices_and_context_window.json",
		"https://mirror.example/catalog.json",
	} {
		cfg := Config{Pricing: PricingConfig{RemoteURL: source, HashURL: "old-hash", OverrideFile: "custom.json"}}
		cfg.normalizePricingCatalogSource()
		require.Equal(t, source, cfg.Pricing.RemoteURL)
		require.Equal(t, "old-hash", cfg.Pricing.HashURL)
		require.Equal(t, "custom.json", cfg.Pricing.OverrideFile)
	}
	cfg := Config{Pricing: PricingConfig{CatalogFormat: "models_dev", RemoteURL: "https://models.dev/catalog.json", HashURL: "old"}}
	cfg.normalizePricingCatalogSource()
	require.Empty(t, cfg.Pricing.HashURL)
}
