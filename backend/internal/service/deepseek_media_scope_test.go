package service

import (
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"testing"
)

// 图片改写不扩散到同用无状态 Responses 规范化的其它平台。
func TestDeepSeekToolMediaRewritePlatformScope(t *testing.T) {
	body := []byte(`{"model":"example","input":[{"type":"function_call_output","call_id":"c","output":[{"type":"input_image","image_url":"data:image/png;base64,AQ=="}]}]}`)
	for _, platform := range []string{PlatformDeepseek, PlatformKimi, PlatformMiniMax, PlatformOpenAI} {
		a := &Account{Platform: platform, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_protocol": "responses"}}
		out := normalizeDeepSeekResponsesRequestBody(a, body)
		if platform == PlatformDeepseek {
			require.Equal(t, int64(2), gjson.GetBytes(out, "input.#").Int())
			require.Equal(t, "user", gjson.GetBytes(out, "input.1.role").String())
		} else {
			require.Equal(t, int64(1), gjson.GetBytes(out, "input.#").Int())
			require.True(t, gjson.GetBytes(out, "input.0.output").IsArray())
		}
	}
}
