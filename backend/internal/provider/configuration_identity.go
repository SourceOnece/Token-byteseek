package provider

import (
	"net/url"
	"strings"
)

// IsOllamaCloudUsageRecord只识别既有官方API Key身份，不依据客户端声称的品牌。
func IsOllamaCloudUsageRecord(r *Record) bool {
	if r == nil || r.Type != "apikey" || (r.Platform != "openai" && r.Platform != "anthropic") {
		return false
	}
	base, _ := r.Credentials["base_url"].(string)
	return IsOllamaCloudBaseURL(base)
}

// IsOllamaCloudBaseURL限定原官方域名、HTTPS、端口和路径。
func IsOllamaCloudBaseURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, "?#") {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Opaque != "" || !strings.EqualFold(parsed.Scheme, "https") || parsed.User != nil || parsed.ForceQuery || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawFragment != "" {
		return false
	}
	hostname := strings.ToLower(parsed.Hostname())
	if hostname != "ollama.com" && hostname != "www.ollama.com" {
		return false
	}
	authority := strings.ToLower(parsed.Host)
	if authority != hostname && authority != hostname+":443" {
		return false
	}
	if parsed.RawPath != "" {
		return false
	}
	return parsed.Path == "" || parsed.Path == "/v1"
}

// ShouldEnsureFingerprintSeed仅在原显式收敛模式开启时要求保留/生成seed。
func ShouldEnsureFingerprintSeed(extra map[string]any) bool {
	mode, _ := extra["codex_fingerprint_mode"].(string)
	switch strings.TrimSpace(mode) {
	case "device", "session", "full":
		return true
	}
	return false
}
