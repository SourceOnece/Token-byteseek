package config

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPricingCatalogLegacySourceMigration 验证旧公共源迁移及自定义源隔离。
func TestPricingCatalogLegacySourceMigration(t *testing.T) {
	for _, source := range []string{
		"https://raw.githubusercontent.com/Wei-Shaw/model-price-repo/main/model_prices_and_context_window.json",
		"https://raw.githubusercontent.com/Wei-Shaw/model-price-repo/refs/heads/main//model_prices_and_context_window.json",
		"https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json",
	} {
		cfg := Config{Pricing: PricingConfig{RemoteURL: source}}
		cfg.normalizePricingCatalogSource()
		cfg.normalizePricingCatalogSource()
		require.Equal(t, "https://models.dev/catalog.json", cfg.Pricing.RemoteURL)
		require.Equal(t, []string{"models.dev"}, cfg.Security.URLAllowlist.PricingHosts)
	}
	for _, source := range []string{
		"https://mirror.example/catalog.json?source=raw.githubusercontent.com/wei-shaw/model-price-repo/main/model_prices_and_context_window.json",
		"https://raw.githubusercontent.com.evil.example/BerriAI/litellm/main/model_prices_and_context_window.json",
		"https://raw.githubusercontent.com/custom/litellm/main/model_prices_and_context_window.json",
		"https://raw.githubusercontent.com/BerriAI/litellm-other/main/model_prices_and_context_window.json",
		"",
	} {
		cfg := Config{Pricing: PricingConfig{RemoteURL: source}}
		cfg.normalizePricingCatalogSource()
		require.Equal(t, source, cfg.Pricing.RemoteURL)
	}
}

// TestPricingCatalogConfigAliases 通过完整加载入口验证别名、环境优先级及退役键不再校验。
func TestPricingCatalogConfigAliases(t *testing.T) {
	for _, tc := range []struct {
		name, yaml, oldEnv, newEnv string
		want                       int
	}{
		{"default", "", "", "", 10},
		{"old yaml", "  hash_check_interval_minutes: 17\n", "", "", 17},
		{"new yaml", "  hash_check_interval_minutes: 17\n  check_interval_minutes: 18\n", "", "", 18},
		{"old env", "  check_interval_minutes: 18\n", "19", "", 19},
		{"new env", "  check_interval_minutes: 18\n", "19", "20", 20},
		{"zero", "  check_interval_minutes: 0\n", "", "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := "pricing:\n  hash_url: [obsolete]\n  update_interval_hours: obsolete\n" + tc.yaml
			file := prepareLegacyConfigTest(t, body)
			t.Setenv("PRICING_HASH_CHECK_INTERVAL_MINUTES", tc.oldEnv)
			t.Setenv("PRICING_CHECK_INTERVAL_MINUTES", tc.newEnv)
			t.Setenv("PRICING_HASH_URL", "obsolete")
			t.Setenv("PRICING_UPDATE_INTERVAL_HOURS", "obsolete")
			cfg, err := Load()
			require.NoError(t, err)
			require.Equal(t, tc.want, cfg.Pricing.CheckIntervalMinutes)
			saved, err := os.ReadFile(file)
			require.NoError(t, err)
			require.Equal(t, body, string(saved))
		})
	}
}

// TestPricingCatalogPackagedFallbackMigration 只迁移缺失的已知打包路径，不替换管理员现有文件。
func TestPricingCatalogPackagedFallbackMigration(t *testing.T) {
	t.Chdir(t.TempDir())
	old := "./resources/model-pricing/model_prices_and_context_window.json"
	cfg := Config{Pricing: PricingConfig{FallbackFile: old}}
	cfg.normalizePricingCatalogSource()
	require.Equal(t, "resources/model-pricing/model_pricing_supplements.json", cfg.Pricing.FallbackFile)
	require.NoError(t, os.MkdirAll(filepath.Dir(old), 0o700))
	require.NoError(t, os.WriteFile(old, []byte(`{}`), 0o600))
	cfg.Pricing.FallbackFile = old
	cfg.normalizePricingCatalogSource()
	require.Equal(t, old, cfg.Pricing.FallbackFile)
	cfg.Pricing.FallbackFile = "./custom/model_prices_and_context_window.json"
	cfg.normalizePricingCatalogSource()
	require.Equal(t, "./custom/model_prices_and_context_window.json", cfg.Pricing.FallbackFile)
}

// TestPricingCatalogSupplementConfig 验证默认不依赖外部文件，自定义路径及环境变量优先级保持有效。
func TestPricingCatalogSupplementConfig(t *testing.T) {
	for _, tc := range []struct {
		name, yaml, env, want string
	}{
		{"default", "", "", ""},
		{"custom yaml", "  fallback_file: ./custom/prices.json\n", "", "./custom/prices.json"},
		{"custom env", "  fallback_file: ./custom/prices.json\n", "/data/custom.json", "/data/custom.json"},
		{"explicit resource", "  fallback_file: ./resources/model-pricing/model_pricing_supplements.json\n", "", "./resources/model-pricing/model_pricing_supplements.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := "pricing:\n  remote_url: https://models.dev/catalog.json\n" + tc.yaml
			file := prepareLegacyConfigTest(t, body)
			t.Setenv("PRICING_FALLBACK_FILE", tc.env)
			cfg, err := Load()
			require.NoError(t, err)
			require.Equal(t, tc.want, cfg.Pricing.FallbackFile)
			saved, err := os.ReadFile(file)
			require.NoError(t, err)
			require.Equal(t, body, string(saved))
		})
	}
}

// TestRetiredPricingOverrideWarnsWithoutRewritingFiles 验证旧键只提示迁移，既不加载也不改写文件。
func TestRetiredPricingOverrideWarnsWithoutRewritingFiles(t *testing.T) {
	for _, env := range []bool{false, true} {
		t.Run(fmt.Sprint(env), func(t *testing.T) {
			legacy := filepath.Join(t.TempDir(), "old-prices.json")
			require.NoError(t, os.WriteFile(legacy, []byte("invalid retired file"), 0o600))
			body := "pricing:\n  remote_url: https://models.dev/catalog.json\n"
			if !env {
				body += "  override_file: " + legacy + "\n"
			}
			file := prepareLegacyConfigTest(t, body)
			value := ""
			if env {
				value = legacy
			}
			t.Setenv("PRICING_OVERRIDE_FILE", value)
			var logs bytes.Buffer
			previous := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
			t.Cleanup(func() { slog.SetDefault(previous) })
			cfg, err := Load()
			require.NoError(t, err)
			require.Equal(t, "https://models.dev/catalog.json", cfg.Pricing.RemoteURL)
			require.Contains(t, logs.String(), "pricing.override_file is retired and ignored")
			saved, err := os.ReadFile(file)
			require.NoError(t, err)
			require.Equal(t, body, string(saved))
			data, err := os.ReadFile(legacy)
			require.NoError(t, err)
			require.Equal(t, "invalid retired file", string(data))
		})
	}
}
