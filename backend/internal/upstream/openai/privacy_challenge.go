package openai

import "strings"

// 隐私设置单独识别挑战页，维持既有 403/503 诊断语义。
func isCloudflareChallengeResponse(cfMitigated, body string) bool {
	return strings.EqualFold(strings.TrimSpace(cfMitigated), "challenge") || strings.Contains(body, "cloudflare") || strings.Contains(body, "cf-") || strings.Contains(body, "Just a moment")
}
