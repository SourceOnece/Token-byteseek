package openai

import (
	"encoding/base64"
	"encoding/json"
	"net/mail"
	"strings"
)

// TokenDisplayEmail 只为管理展示解码邮箱，不验证签名，不用于鉴权、账号归属或调度决策。
func TokenDisplayEmail(token string) string {
	if len(token) > 128*1024 {
		return ""
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return ""
	}
	var claims struct {
		Email   string `json:"email"`
		Profile struct {
			Email string `json:"email"`
		} `json:"https://api.openai.com/profile"`
	}
	if json.Unmarshal(raw, &claims) != nil {
		return ""
	}
	for _, value := range []string{claims.Email, claims.Profile.Email} {
		value = strings.TrimSpace(value)
		parsed, err := mail.ParseAddress(value)
		if err == nil && parsed.Address == value && len(value) <= 254 {
			return value
		}
	}
	return ""
}
