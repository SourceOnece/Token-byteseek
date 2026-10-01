package openai

import (
	"testing"

	"github.com/tidwall/gjson"
)

// 核对本地 OAuth 归一化是否保留新型号的 mode 以及去除不支持的采样字段。
func TestGPT61ReasoningModeAndSampling(t *testing.T) {
	body := []byte(`{"model":"gpt-6.1-sol","reasoning":{"mode":"pro","effort":"max"},"temperature":0.5,"top_p":0.9,"logprobs":true,"top_logprobs":2,"include":["message.output_text.logprobs","reasoning.encrypted_content"]}`)
	out, _, err := NormalizeOpenAIPassthroughOAuthBody(body, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("mode=%s temperature=%v logprobs=%v include=%s", gjson.GetBytes(out, "reasoning.mode"), gjson.GetBytes(out, "temperature").Exists(), gjson.GetBytes(out, "logprobs").Exists(), gjson.GetBytes(out, "include").Raw)
	if gjson.GetBytes(out, "reasoning.mode").String() != "pro" {
		t.Error("new model pro mode removed")
	}
	for _, k := range []string{"temperature", "top_p", "logprobs", "top_logprobs"} {
		if gjson.GetBytes(out, k).Exists() {
			t.Error("unsupported field preserved", k)
		}
	}
}
